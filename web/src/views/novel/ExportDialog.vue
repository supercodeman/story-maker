<!-- web/src/views/novel/ExportDialog.vue -->
<template>
  <el-dialog v-model="visible" title="导出小说" width="420px">
    <el-tabs v-model="activeTab">
      <el-tab-pane label="Word 文档" name="word">
        <p style="color: var(--el-text-color-secondary); font-size: 13px; margin-bottom: 16px">
          导出全本小说为 Word 文档，包含封面、目录和图文混排章节内容。
        </p>
      </el-tab-pane>
      <el-tab-pane label="全本音频" name="audio">
        <p style="color: var(--el-text-color-secondary); font-size: 13px; margin-bottom: 12px">
          将所有章节转换为音频并打包为 ZIP，包含分章节 MP3 和全本合并音频。
        </p>
        <el-form label-width="60px" size="small">
          <el-form-item label="音色">
            <el-select v-model="voiceId" style="width: 100%">
              <el-option label="御姐" value="female-yujie" />
              <el-option label="少女" value="female-shaonv" />
              <el-option label="磁性男声" value="male-qn-qingse" />
              <el-option label="沉稳男声" value="male-qn-jingying" />
            </el-select>
          </el-form-item>
          <el-form-item label="语速">
            <el-input-number
              v-model="speed"
              :min="0.5"
              :max="2.0"
              :step="0.1"
              :precision="1"
              controls-position="right"
              style="width: 100%"
            />
          </el-form-item>
        </el-form>
      </el-tab-pane>
    </el-tabs>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleExport">
        开始导出
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Document, Packer, Paragraph, TextRun, HeadingLevel, AlignmentType, PageBreak } from 'docx'
import { saveAs } from 'file-saver'
import { mediaApi } from '@/api/media'

const props = defineProps<{
  novelId: number
}>()

const visible = defineModel<boolean>({ default: false })
const activeTab = ref('word')
const voiceId = ref('female-yujie')
const speed = ref(1.0)
const loading = ref(false)

function buildDocx(data: { title: string; description: string; chapters: { title: string; content: string }[] }) {
  const children: Paragraph[] = []

  // 封面
  children.push(new Paragraph({ alignment: AlignmentType.CENTER, spacing: { before: 3000 }, children: [new TextRun({ text: data.title, bold: true, size: 56, font: 'Microsoft YaHei' })] }))
  if (data.description) {
    children.push(new Paragraph({ alignment: AlignmentType.CENTER, spacing: { before: 400 }, children: [new TextRun({ text: data.description, size: 24, color: '666666', font: 'Microsoft YaHei' })] }))
  }
  children.push(new Paragraph({ children: [new PageBreak()] }))

  // 章节
  for (const ch of data.chapters) {
    children.push(new Paragraph({ heading: HeadingLevel.HEADING_1, spacing: { before: 360, after: 120 }, children: [new TextRun({ text: ch.title, bold: true, size: 36, font: 'Microsoft YaHei' })] }))
    const paragraphs = ch.content.split('\n')
    for (const p of paragraphs) {
      const text = p.trim()
      if (text) {
        children.push(new Paragraph({ spacing: { line: 400 }, indent: { firstLine: 480 }, children: [new TextRun({ text, size: 24, font: 'Microsoft YaHei' })] }))
      }
    }
    children.push(new Paragraph({ children: [new PageBreak()] }))
  }

  return new Document({ sections: [{ children }] })
}

async function handleExport() {
  loading.value = true
  try {
    if (activeTab.value === 'word') {
      const res: any = await mediaApi.exportWord(props.novelId)
      const doc = buildDocx(res)
      const blob = await Packer.toBlob(doc)
      saveAs(blob, `${res.title || '小说导出'}.docx`)
      ElMessage.success('Word 文档已下载')
    } else {
      await mediaApi.exportAudio(props.novelId, {
        voice_id: voiceId.value,
        speed: speed.value,
      })
      ElMessage.success('音频导出任务已提交，完成后将通过通知推送下载链接')
    }
    visible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '导出失败')
  } finally {
    loading.value = false
  }
}
</script>
