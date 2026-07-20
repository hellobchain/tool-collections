<template>
  <div class="contract-draft">
    <div class="page-header">
      <h2>✍️ 合同起草助手</h2>
      <p class="page-desc">上传合同模板并输入需求，自动生成定制化合同草案</p>
    </div>

    <el-steps :active="activeStep" align-center class="draft-steps">
      <el-step title="上传模板" description="doc/docx" />
      <el-step title="输入需求" description="自然语言" />
      <el-step title="生成结果" description="预览与导出" />
    </el-steps>

    <!-- Step 1: Upload template -->
    <div v-show="activeStep === 0" class="step-content">
      <el-card class="upload-card">
        <el-upload
          ref="upload"
          drag
          action=""
          :auto-upload="false"
          :show-file-list="false"
          :on-change="handleFileChange"
          :on-remove="handleFileRemove"
          :file-list="fileList"
          :accept="'.doc,.docx'"
        >
          <i class="el-icon-upload"></i>
          <div class="el-upload__text">拖拽合同模板到此处，或<em>点击选择</em></div>
          <div slot="tip" class="el-upload__tip">支持 .doc .docx 格式，单文件不超过10MB</div>
        </el-upload>
      </el-card>
      <el-card v-if="fileList.length" class="file-list-card">
        <div v-for="f in fileList" :key="f.id" class="file-item">
          <span class="file-name">{{ f.name }}</span>
          <span class="file-size">{{ toKB(f.size) }}</span>
          <el-button type="text" size="mini" icon="el-icon-delete" @click="removeFile(f)" style="color:#999;"/>
        </div>
      </el-card>

      <el-card class="req-card">
        <div slot="header">📋 输入合同需求</div>
        <el-input
          v-model="requirements"
          type="textarea"
          :rows="6"
          placeholder="请输入合同关键要素，例如：&#10;合同类型：软件外包开发合同&#10;甲方：北京XX科技有限公司&#10;乙方：上海YY软件开发有限公司&#10;项目内容：开发一套ERP管理系统&#10;合同金额：50万元&#10;付款方式：预付30%，验收后付70%&#10;交付期限：合同签订后90天内&#10;特殊要求：源代码归甲方所有，乙方需提供1年免费质保"
        />
        <div class="req-count">{{ requirements.length }} / 500</div>
      </el-card>

      <div class="step-actions">
        <el-button :disabled="!canProceedStep1" type="primary" @click="handleStartDraft">开始生成 →</el-button>
      </div>
    </div>

    <!-- Step 2: Generating -->
    <div v-show="activeStep === 1" class="step-content">
      <el-card class="progress-card">
        <div class="progress-info">
          <span>当前进度：{{ progressText }}</span>
        </div>
        <el-progress :percentage="progressPercent" :status="progressPercent >= 100 ? 'success' : undefined" />
        <div class="progress-detail" v-if="currentStep">
          <i class="el-icon-loading"></i> {{ currentStep }}
        </div>
      </el-card>
    </div>

    <!-- Step 3: Result -->
    <div v-show="activeStep === 2" class="step-content">
      <el-card class="result-card">
        <div class="result-header">
          <span class="result-title">📄 合同草案</span>
          <div class="result-actions">
            <el-button type="primary" icon="el-icon-download" @click="handleDownload">下载 .docx</el-button>
            <el-button @click="handleBack">返回重新生成</el-button>
          </div>
        </div>
        <div class="result-meta">
          <span>模板：{{ fileName }}</span>
          <span>生成时间：{{ generateTime }}</span>
        </div>
        <div class="result-preview">
          <div class="draftContent-md" v-html="renderMd(draftContent)"></div>
        </div>
      </el-card>

      <el-card class="changelog-card" v-if="changeLog">
        <div slot="header">📝 条款变更说明</div>
        <div class="changelog-md" v-html="renderMd(changeLog)"></div>
      </el-card>
    </div>
  </div>
</template>

<script>
import { marked } from 'marked'
import { uploadContract, startDraftGen, getDraftProgress, getDraftResult, downloadDraft } from '@/api/contract'

export default {
  name: 'ContractDraft',
  data() {
    return {
      activeStep: 0,
      fileList: [],
      selectedFile: null,
      fileName: '',
      requirements: '',
      pollTimer: null,
      progressPercent: 0,
      progressText: '',
      currentStep: '',
      taskId: null,
      draftContent: '',
      changeLog: '',
      generateTime: ''
    }
  },
  computed: {
    canProceedStep1() {
      return this.selectedFile && this.requirements.trim().length > 0
    }
  },
  beforeDestroy() {
    this.stopPolling()
  },
  methods: {
    toKB(bytes) {
      if (!bytes) return '0 KB'
      return (bytes / 1024).toFixed(2) + ' KB'
    },
    removeFile(file) {
      this.fileList = this.fileList.filter(f => f.id !== file.id)
      if (this.selectedFile && this.selectedFile.name === file.name) {
        this.selectedFile = null
        this.fileName = ''
      }
    },
    renderMd(text) {
      if (!text) return ''
      return marked.parse(text, { breaks: true })
    },
    handleFileRemove() {
      this.selectedFile = null
      this.fileList = []
    },
    handleFileChange(file) {
      if (!/\.docx?$/i.test(file.name)) {
        this.$message.error('仅支持 .doc .docx 格式')
        return
      }
      if (file.size > 10 * 1024 * 1024) {
        this.$message.error('文件大小不能超过 10MB')
        return
      }
      if (this.fileList.length > 0) {
        this.$message.error('仅支持单个文件')
        return;
      }
      this.fileList = [file]
      this.selectedFile = file.raw
      this.fileName = file.name
    },
    async handleStartDraft() {
      this.activeStep = 1
      this.progressPercent = 0
      this.progressText = '上传模板中...'
      this.currentStep = '正在上传合同模板'

      try {
        const res = await uploadContract(this.selectedFile, p => {
          const pct = Math.round((p.loaded / p.total) * 100)
          this.progressPercent = pct
        })
        const serverFile = res.data.data || res.data

        this.progressText = 'AI分析中...'
        this.currentStep = '正在解析模板结构和需求'

        const draftRes = await startDraftGen({
          file_id: serverFile.id,
          requirements: this.requirements
        })
        const { task_id } = draftRes.data.data || draftRes.data
        this.taskId = task_id
        this.startPolling()
      } catch (e) {
        this.activeStep = 0
      }
    },
    startPolling() {
      this.pollTimer = setInterval(async () => {
        try {
          const res = await getDraftProgress(this.taskId)
          const data = res.data.data || res.data
          this.progressPercent = data.percent || 0
          this.currentStep = data.current_step || ''
          this.progressText = `AI处理中 ${this.progressPercent}%`

          if (data.percent >= 100 || data.status === 'completed') {
            this.stopPolling()
            this.fetchResult()
          } else if (data.status === 'failed') {
            this.stopPolling()
            this.$message.error('生成失败')
            this.activeStep = 0
            return
          }
        } catch {
          this.stopPolling()
        }
      }, 4000)
    },
    stopPolling() {
      if (this.pollTimer) { clearInterval(this.pollTimer); this.pollTimer = null }
    },
    async fetchResult() {
      try {
        const res = await getDraftResult(this.taskId)
        const data = res.data.data || res.data
        this.draftContent = data.content || ''
        this.changeLog = data.change_log || ''
        this.generateTime = data.generated_at || new Date().toLocaleString()
        this.activeStep = 2
      } catch {
      }
    },
    async handleDownload() {
      try {
        const res = await downloadDraft(this.taskId)
        const blob = res.data
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `合同草案_${this.fileName.replace(/\.docx?$/i, '')}.docx`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)
      } catch {
      }
    },
    handleBack() {
      this.activeStep = 0
      this.draftContent = ''
      this.changeLog = ''
      this.taskId = null
      this.progressPercent = 0
    }
  }
}
</script>

<style scoped>
.contract-draft {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: auto;
}
.page-header {
  background: #fff;
  padding: 8px 20px;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}
.file-item { display: flex; align-items: center; justify-content: space-between; padding: 4px 0; font-size: 13px; color: #333; }
.file-name { flex: 1; }
.file-size { flex: 1; color: #999; font-size: 12px; text-align: left; }
.file-list-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.file-list-card >>> .el-card__body { padding: 6px 16px; }
.page-header h2 { font-size: 16px; margin: 0; color: #333; }
.page-desc { display: none; }
.draft-steps { padding: 10px 20px; background: #fff; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; }
.step-content { flex: 1; padding: 0; overflow: auto; }
.upload-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.upload-card >>> .el-upload-dragger { margin-bottom: 0; width: 100%; }
.upload-card >>> .el-card__body { padding: 12px 16px; }
.upload-card >>> .el-upload { width: 100%; }
.upload-card >>> .el-upload-dragger { width: 100%; padding: 16px; }
.req-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.req-card >>> .el-card__body { padding: 8px 16px; }
.req-count { text-align: right; font-size: 12px; color: #999; margin-top: 4px; }
.step-actions { display: flex; gap: 8px; justify-content: center; padding: 8px 0; }
.progress-card { max-width: 600px; margin: 0 auto; border-top: none; border-radius: 0; }
.progress-info { margin-bottom: 6px; font-size: 14px; color: #666; }
.progress-detail { margin-top: 6px; font-size: 13px; color: #409eff; }
.result-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.result-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.result-title { font-size: 16px; font-weight: 600; }
.result-actions { display: flex; gap: 8px; }
.result-meta { display: flex; gap: 20px; font-size: 13px; color: #999; margin-bottom: 12px; }
.result-preview { border: 1px solid #e4e7ed; border-radius: 4px; max-height: 60vh; overflow: auto; }
.preview-content { white-space: pre-wrap; word-break: break-all; line-height: 1.8; font-size: 13px; padding: 16px; margin: 0; }
.changelog-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.changelog-content { white-space: pre-wrap; font-size: 13px; line-height: 1.6; margin: 0; }
</style>
