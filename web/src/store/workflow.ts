// web/src/store/workflow.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { workflowApi } from '@/api/workflow'
import type { Workflow, WorkflowNode } from '@/api/workflow'

// 判断工作流是否处于终态（含 completed_with_warning）
function isTerminal(status: string): boolean {
  return status === 'completed' || status === 'completed_with_warning' || status === 'failed' || status === 'cancelled'
}

// 判断工作流是否成功完成（含 completed_with_warning）
export function isCompleted(status: string): boolean {
  return status === 'completed' || status === 'completed_with_warning'
}

export const useWorkflowStore = defineStore('workflow', () => {
  const currentWorkflow = ref<Workflow | null>(null)
  const nodes = ref<WorkflowNode[]>([])
  const pending = ref(false)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  const progress = computed(() => {
    if (!currentWorkflow.value || currentWorkflow.value.total_nodes === 0) return 0
    return currentWorkflow.value.completed_nodes / currentWorkflow.value.total_nodes
  })

  // 启动轮询 fallback（提取公共逻辑，submitWorkflow 和 recoverFromWorkflow 共用）
  function startPolling() {
    stopPolling()
    pollTimer = setInterval(async () => {
      if (!currentWorkflow.value) { stopPolling(); return }
      try {
        const detail: any = await workflowApi.get(currentWorkflow.value.id)
        const wf = detail.workflow || detail
        if (isTerminal(wf.status)) {
          stopPolling()
          handleWorkflowUpdate({
            id: wf.id,
            status: wf.status,
            completed_nodes: wf.completed_nodes,
            total_nodes: wf.total_nodes,
            error_msg: wf.error_msg,
            result_json: wf.result_json,
          })
          if (detail.nodes) nodes.value = detail.nodes
        } else {
          // 更新进度
          if (wf.completed_nodes !== undefined) {
            currentWorkflow.value.completed_nodes = wf.completed_nodes
            currentWorkflow.value.total_nodes = wf.total_nodes
          }
          // 同步节点列表（恢复场景下首次轮询可填充节点）
          if (detail.nodes) nodes.value = detail.nodes
        }
      } catch { /* 轮询失败忽略 */ }
    }, 3000)
  }

  // 提交工作流
  async function submitWorkflow(
    portfolioId: number,
    workflowType: string,
    modelName: string,
    params: Record<string, any>,
  ) {
    pending.value = true
    try {
      const data: any = await workflowApi.submit({
        portfolio_id: portfolioId,
        workflow_type: workflowType,
        model_name: modelName,
        params,
      })
      currentWorkflow.value = {
        id: data.workflow_id,
        user_id: 0,
        portfolio_id: portfolioId,
        workflow_type: workflowType,
        status: 'pending',
        total_nodes: 0,
        completed_nodes: 0,
        result_json: '',
        error_msg: '',
        initial_context: JSON.stringify(params),
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }
      nodes.value = []
      startPolling()
      return data.workflow_id
    } catch (e) {
      pending.value = false
      throw e
    }
  }

  // 处理工作流整体更新（WebSocket 回调）
  function handleWorkflowUpdate(data: {
    id: number
    status: string
    completed_nodes: number
    total_nodes: number
    error_msg?: string
    result_json?: string
  }) {
    if (!currentWorkflow.value || currentWorkflow.value.id !== data.id) return

    currentWorkflow.value.status = data.status
    currentWorkflow.value.completed_nodes = data.completed_nodes
    currentWorkflow.value.total_nodes = data.total_nodes

    if (data.error_msg) {
      currentWorkflow.value.error_msg = data.error_msg
    }
    if (data.result_json) {
      currentWorkflow.value.result_json = data.result_json
    }

    // 终态时关闭 pending 和轮询
    if (isTerminal(data.status)) {
      pending.value = false
      stopPolling()
    }
  }

  // 处理节点级更新（WebSocket 回调）
  function handleNodeUpdate(data: {
    workflow_id: number
    node_id: string
    status: string
    result_json?: any
    error_msg?: string
  }) {
    if (!currentWorkflow.value || currentWorkflow.value.id !== data.workflow_id) return

    // WebSocket 推送的 result_json 可能是 object，统一转为 string
    const resultStr = data.result_json
      ? (typeof data.result_json === 'string' ? data.result_json : JSON.stringify(data.result_json))
      : ''

    const idx = nodes.value.findIndex((n) => n.node_id === data.node_id)
    if (idx !== -1) {
      nodes.value[idx].status = data.status
      if (resultStr) nodes.value[idx].result_json = resultStr
      if (data.error_msg) nodes.value[idx].error_msg = data.error_msg
    } else {
      nodes.value.push({
        id: 0,
        workflow_id: data.workflow_id,
        node_id: data.node_id,
        task_id: 0,
        status: data.status,
        result_json: resultStr,
        error_msg: data.error_msg || '',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      })
    }
  }

  // 查询工作流详情
  async function fetchWorkflow(id: number) {
    const data: any = await workflowApi.get(id)
    currentWorkflow.value = data.workflow
    nodes.value = data.nodes || []
    pending.value = data.workflow.status === 'pending' || data.workflow.status === 'running'
  }

  // 取消工作流
  async function cancelWorkflow(id: number) {
    await workflowApi.cancel(id)
    if (currentWorkflow.value?.id === id) {
      currentWorkflow.value.status = 'cancelled'
      pending.value = false
      stopPolling()
    }
  }

  // 从活跃工作流恢复状态（页面刷新后调用）
  function recoverFromWorkflow(workflow: Workflow) {
    currentWorkflow.value = workflow
    nodes.value = []
    pending.value = true
    // 立即 fetch 一次详情，填充节点列表和最新进度
    workflowApi.get(workflow.id).then((detail: any) => {
      const wf = detail.workflow || detail
      if (currentWorkflow.value && currentWorkflow.value.id === workflow.id) {
        currentWorkflow.value.completed_nodes = wf.completed_nodes
        currentWorkflow.value.total_nodes = wf.total_nodes
        if (detail.nodes) nodes.value = detail.nodes
      }
    }).catch(() => { /* 忽略 */ })
    startPolling()
  }

  function reset() {
    stopPolling()
    currentWorkflow.value = null
    nodes.value = []
    pending.value = false
  }

  return {
    currentWorkflow,
    nodes,
    pending,
    progress,
    submitWorkflow,
    handleWorkflowUpdate,
    handleNodeUpdate,
    fetchWorkflow,
    cancelWorkflow,
    recoverFromWorkflow,
    reset,
  }
})
