<!-- web/src/views/novel/PlotTimeline.vue -->
<template>
  <div class="plot-timeline">
    <div class="plot-timeline__banner">
      <span class="plot-timeline__banner-icon">📖</span>
      <div class="plot-timeline__banner-text">
        <div class="plot-timeline__banner-title">情节线</div>
        <div class="plot-timeline__banner-desc">按时间顺序梳理小说的主要情节脉络，关联对应章节，帮助你把控故事节奏和结构完整性。支持 AI 自动提取或手动添加。</div>
      </div>
      <el-button size="small" type="primary" text @click="handleAdd">+ 新增情节线</el-button>
    </div>

    <div v-if="overviewStore.plotlines.length === 0" class="plot-timeline__empty">
      暂无情节线数据，可通过 AI 提取或手动添加
    </div>

    <div class="plot-timeline__list">
      <div
        v-for="(item, index) in sortedPlotlines"
        :key="item.id"
        class="plot-node"
        :class="{ 'plot-node--editing': editingId === item.id }"
      >
        <div class="plot-node__marker">
          <div class="plot-node__dot" />
          <div v-if="index < sortedPlotlines.length - 1" class="plot-node__line" />
        </div>

        <div class="plot-node__content">
          <template v-if="editingId === item.id">
            <el-input v-model="editForm.title" size="small" placeholder="情节线标题" />
            <el-input
              v-model="editForm.content"
              type="textarea"
              :rows="3"
              size="small"
              placeholder="情节描述"
              style="margin-top: 6px;"
            />
            <el-input
              v-model="editForm.chapter_ref"
              size="small"
              placeholder="关联章节（逗号分隔，如 1,3,5）"
              style="margin-top: 6px;"
            />
            <div class="plot-node__edit-actions">
              <el-button size="small" type="primary" @click="handleSave(item)">保存</el-button>
              <el-button size="small" @click="editingId = null">取消</el-button>
            </div>
          </template>
          <template v-else>
            <div class="plot-node__header">
              <span class="plot-node__title">{{ item.title }}</span>
              <div class="plot-node__actions">
                <el-button size="small" text @click="handleEdit(item)">编辑</el-button>
                <el-button size="small" text type="danger" @click="handleDelete(item)">删除</el-button>
              </div>
            </div>
            <p class="plot-node__desc">{{ item.content }}</p>
            <div v-if="item.chapter_ref" class="plot-node__chapters">
              <el-tag
                v-for="ch in parseChapterRefs(item.chapter_ref)"
                :key="ch"
                size="small"
                type="info"
              >
                Ch.{{ ch }}
              </el-tag>
            </div>
          </template>
        </div>
      </div>
    </div>

    <!-- 新增对话框 -->
    <el-dialog v-model="showAddDialog" title="新增情节线" width="500px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item label="标题">
          <el-input v-model="addForm.title" placeholder="情节线标题" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="addForm.content" type="textarea" :rows="4" placeholder="情节描述" />
        </el-form-item>
        <el-form-item label="关联章节">
          <el-input v-model="addForm.chapter_ref" placeholder="逗号分隔，如 1,3,5" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useOverviewStore } from '@/store/overview'
import { knowledgeApi } from '@/api/knowledge'
import type { NovelKnowledge } from '@/api/knowledge'
import { ElMessage, ElMessageBox } from 'element-plus'

const props = defineProps<{ novelId: number }>()
const overviewStore = useOverviewStore()

const editingId = ref<number | null>(null)
const editForm = ref({ title: '', content: '', chapter_ref: '' })
const showAddDialog = ref(false)
const addForm = ref({ title: '', content: '', chapter_ref: '' })

const sortedPlotlines = computed(() =>
  [...overviewStore.plotlines].sort((a, b) => a.sort_order - b.sort_order)
)

function parseChapterRefs(refs: string): string[] {
  return refs.split(',').map(s => s.trim()).filter(Boolean)
}

function handleAdd() {
  addForm.value = { title: '', content: '', chapter_ref: '' }
  showAddDialog.value = true
}

async function handleCreate() {
  if (!addForm.value.title) {
    ElMessage.warning('请输入标题')
    return
  }
  try {
    const item: any = await knowledgeApi.create(props.novelId, {
      category: 'plotline',
      title: addForm.value.title,
      content: addForm.value.content,
      chapter_ref: addForm.value.chapter_ref,
    })
    overviewStore.plotlines.push(item)
    overviewStore.addChange({
      type: 'plotline',
      action: 'create',
      data: item,
    })
    showAddDialog.value = false
    ElMessage.success('创建成功')
  } catch {
    ElMessage.error('创建失败')
  }
}

function handleEdit(item: NovelKnowledge) {
  editingId.value = item.id
  editForm.value = {
    title: item.title,
    content: item.content,
    chapter_ref: item.chapter_ref,
  }
}

async function handleSave(item: NovelKnowledge) {
  try {
    const oldData = { title: item.title, content: item.content, chapter_ref: item.chapter_ref }
    const updated: any = await knowledgeApi.update(item.id, {
      title: editForm.value.title,
      content: editForm.value.content,
      chapter_ref: editForm.value.chapter_ref,
    })
    const idx = overviewStore.plotlines.findIndex(p => p.id === item.id)
    if (idx >= 0) overviewStore.plotlines[idx] = updated
    overviewStore.addChange({
      type: 'plotline',
      action: 'update',
      id: item.id,
      data: updated,
      old_data: oldData,
    })
    editingId.value = null
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存失败')
  }
}

async function handleDelete(item: NovelKnowledge) {
  try {
    await ElMessageBox.confirm('确定删除该情节线？', '确认')
    await knowledgeApi.delete(item.id)
    overviewStore.plotlines = overviewStore.plotlines.filter(p => p.id !== item.id)
    overviewStore.addChange({
      type: 'plotline',
      action: 'delete',
      id: item.id,
      old_data: item,
    })
    ElMessage.success('删除成功')
  } catch {
    // 用户取消或删除失败
  }
}
</script>

<style scoped lang="scss">
.plot-timeline {
  padding: 0;

  &__banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: var(--el-color-primary-light-9);
    border-radius: 6px;
    margin-bottom: 8px;

    .el-button { margin-left: auto; flex-shrink: 0; }
  }

  &__banner-text { flex: 1; min-width: 0; }

  &__banner-icon {
    font-size: 18px;
    line-height: 1;
    flex-shrink: 0;
  }

  &__banner-title {
    font-weight: 600;
    font-size: 13px;
    color: var(--el-text-color-primary);
    margin-bottom: 2px;
  }

  &__banner-desc {
    font-size: 11px;
    color: var(--el-text-color-secondary);
    line-height: 1.4;
  }

  &__toolbar {
    display: none;
  }

  &__empty {
    text-align: center;
    color: var(--el-text-color-secondary);
    padding: 40px 0;
  }

  &__list {
    position: relative;
  }
}

.plot-node {
  display: flex;
  gap: 16px;
  margin-bottom: 8px;

  &__marker {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 20px;
    flex-shrink: 0;
  }

  &__dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: var(--el-color-primary);
    flex-shrink: 0;
  }

  &__line {
    width: 2px;
    flex: 1;
    background: var(--el-border-color);
    margin-top: 4px;
  }

  &__content {
    flex: 1;
    background: var(--el-bg-color-page);
    border-radius: 8px;
    padding: 12px;
    border: 1px solid var(--el-border-color-lighter);
  }

  &__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  &__title {
    font-weight: 600;
    font-size: 14px;
    color: var(--el-text-color-primary);
  }

  &__actions {
    flex-shrink: 0;
  }

  &__desc {
    margin: 6px 0;
    font-size: 13px;
    color: var(--el-text-color-regular);
    line-height: 1.5;
  }

  &__chapters {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
    margin-top: 6px;
  }

  &__edit-actions {
    margin-top: 8px;
    display: flex;
    gap: 8px;
  }
}
</style>
