<template>
  <div class="extract-page">
    <div class="page-header">
      <h2>🔍 合同要素提取</h2>
      <p class="page-desc">上传合同文档，配置需提取的字段，自动抽取结构化数据</p>
    </div>

    <el-steps :active="activeStep" align-center class="extract-steps">
      <el-step title="上传文档" description="doc/docx" />
      <el-step title="配置字段" description="自定义要素" />
      <el-step title="提取结果" description="查看与校验" />
    </el-steps>

    <!-- Step 1: Upload -->
    <div v-show="activeStep === 0" class="step-content">
      <el-card class="upload-card">
        <el-upload
          ref="upload"
          drag
          multiple
          action=""
          :auto-upload="false"
          :show-file-list="false"
          :on-change="handleFileChange"
          :on-remove="handleFileRemove"
          :file-list="fileList"
          :accept="'.doc,.docx'"
        >
          <i class="el-icon-upload"></i>
          <div class="el-upload__text">拖拽合同文件到此处，或<em>点击选择</em></div>
          <div slot="tip" class="el-upload__tip">支持 .doc .docx 格式，单文件不超过50MB，单次最多1份</div>
        </el-upload>
      </el-card>
      <el-card v-if="uploadedFiles.length" class="file-list-card">
        <div v-for="f in uploadedFiles" :key="f.id" class="file-item">
          <span class="file-name">{{ f.name }}</span>
          <span class="file-size">{{ f.size }}</span>
          <el-tag v-if="f.status==='parsed'" type="success" size="mini">就绪</el-tag>
          <el-tag v-else-if="f.status==='uploading'" type="warning" size="mini">上传中</el-tag>
          <el-tag v-else-if="f.status==='failed'" type="danger" size="mini">失败</el-tag>
          <el-button type="text" size="mini" icon="el-icon-delete" @click="removeFile(f)" style="color:#999" />
        </div>
      </el-card>
      <div class="step-actions">
        <el-button :disabled="!uploadedFiles.length" type="primary" @click="activeStep=1">下一步</el-button>
      </div>
    </div>

    <!-- Step 2: Configure Fields -->
    <div v-show="activeStep === 1" class="step-content">
      <el-card class="field-card">
        <div slot="header">
          <span>配置提取字段</span>
          <el-button size="mini" type="primary" style="float:right" @click="addField">+ 添加字段</el-button>
        </div>
        <div v-for="(f, i) in fields" :key="i" class="field-row">
          <el-input v-model="f.name" placeholder="字段名称（如：甲方公司全称）" size="small" style="width:200px" />
          <el-input v-model="f.description" placeholder="字段描述/提取指引" size="small" style="width:300px" />
          <el-select v-model="f.data_type" placeholder="类型" size="small" style="width:100px">
            <el-option label="文本" value="text" />
            <el-option label="数字" value="number" />
            <el-option label="日期" value="date" />
            <el-option label="金额" value="amount" />
          </el-select>
          <el-checkbox v-model="f.required">必填</el-checkbox>
          <el-checkbox v-model="f.multi">多值</el-checkbox>
          <el-button size="mini" type="danger" icon="el-icon-delete" circle @click="fields.splice(i,1)" />
        </div>
        <div v-if="!fields.length" class="field-empty">暂无字段，请添加或选择预设模板</div>
      </el-card>

      <el-card class="template-card" v-if="presetTemplates.length">
        <div slot="header">预设模板</div>
        <el-tag
          v-for="tpl in presetTemplates"
          :key="tpl.name"
          class="template-tag"
          @click="applyTemplate(tpl)"
        >{{ tpl.name }}</el-tag>
      </el-card>

      <div class="step-actions">
        <el-button @click="activeStep=0">上一步</el-button>
        <el-button :disabled="!fields.length" type="primary" @click="handleStartExtract">开始提取 →</el-button>
      </div>
    </div>

    <!-- Step 3: Results -->
    <div v-show="activeStep === 2" class="step-content">
      <el-card v-if="extracting" class="progress-card">
        <el-progress :percentage="extractProgress" />
        <div class="progress-step">{{ extractStep }}</div>
      </el-card>

      <el-card v-if="resultData" class="result-card">
        <div class="result-toolbar">
          <span class="result-title">提取结果 — {{ resultData.task_name }}</span>
          <el-button size="mini" icon="el-icon-download" @click="handleExport">导出CSV</el-button>
          <el-button size="mini" @click="handleBack">返回</el-button>
        </div>
        <el-table :data="tableData" stripe border size="small" style="width:100%">
          <el-table-column prop="file_name" label="文件名" min-width="160" fixed />
          <el-table-column v-for="f in resultFields" :key="f.name" :label="f.name" min-width="140">
            <template slot-scope="{ row }">
              <el-input
                v-if="editingCell === row.file_id + '_' + f.name"
                v-model="row.data[f.name]"
                size="mini"
                @blur="saveEdit(row, f.name)"
                @keyup.enter.native="saveEdit(row, f.name)"
              />
              <span v-else @dblclick="startEdit(row, f.name)">{{ row.data[f.name] }}</span>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
  </div>
</template>

<script>
import { uploadContract, startExtract, getExtractProgress, getExtractResult, updateExtractCell, exportExtractResult } from '@/api/contract'

const PRESET_FIELDS = [
  { name: '合同编号', description: '合同唯一编号', data_type: 'text', required: false, multi: false },
  { name: '合同名称', description: '合同全称/标题', data_type: 'text', required: true, multi: false },
  { name: '甲方公司全称', description: '甲方（采购方/委托方）公司全称', data_type: 'text', required: true, multi: false },
  { name: '乙方公司全称', description: '乙方（供应方/服务方）公司全称', data_type: 'text', required: true, multi: false },
  { name: '合同总金额', description: '合同总金额，只输出数字', data_type: 'amount', required: false, multi: false },
  { name: '签署日期', description: '合同签署日期，YYYY-MM-DD格式', data_type: 'date', required: false, multi: false },
  { name: '合同有效期', description: '合同有效期限', data_type: 'text', required: false, multi: false },
  { name: '付款方式', description: '付款条款描述', data_type: 'text', required: false, multi: false },
  { name: '违约责任', description: '违约责任条款摘要', data_type: 'text', required: false, multi: false },
]

export default {
  name: 'ContractExtract',
  data() {
    return {
      activeStep: 0,
      fileList: [],
      uploadedFiles: [],
      fields: [],
      presetTemplates: [{ name: '常用合同要素', fields: PRESET_FIELDS }],
      taskId: null,
      extracting: false,
      extractProgress: 0,
      extractStep: '',
      pollTimer: null,
      resultData: null,
      resultFields: [],
      editingCell: '',
    }
  },
  computed: {
    tableData() {
      return this.resultData?.results || []
    }
  },
  beforeDestroy() { this.stopPolling() },
  methods: {
    removeFile(file) {
      this.uploadedFiles = this.uploadedFiles.filter(f => f.id !== file.id)
      this.fileList = this.fileList.filter(f => f.id !== file.id)
    },
    handleFileRemove() { this.uploadedFiles = []; this.fileList = [] },
    async handleFileChange(file) {
      if (!/\.docx?$/i.test(file.name)) { 
        this.$message.error('仅支持 .doc .docx'); 
        return 
      }
      if (file.size > 50*1024*1024) {
         this.$message.error('文件不超过50MB'); 
         return 
      }
      if (this.uploadedFiles.length >= 1) { 
         this.$message.error('仅支持单个文件');
         return 
      }
      const raw = file.raw
      const tempId = '_tmp_' + Date.now()
      this.uploadedFiles.push({ id: tempId, name: raw.name, size: this.formatSize(raw.size), status: 'uploading', raw })
      try {
        const res = await uploadContract(raw)
        const server = res.data.data || res.data
        const idx = this.uploadedFiles.findIndex(f => f.id === tempId)
        if (idx >= 0) this.$set(this.uploadedFiles, idx, { id: server.id, name: server.name, size: server.size, status: 'parsed' })
        this.fileList = this.uploadedFiles.filter(f => f.status === 'parsed' || f.status === 'failed').map(f => ({ name: f.name, size: f.size }))
      } catch {
        const idx = this.uploadedFiles.findIndex(f => f.id === tempId)
        if (idx >= 0) this.$set(this.uploadedFiles, idx, { ...this.uploadedFiles[idx], status: 'failed' })
      }
    },
    addField() {
      this.fields.push({ name: '', description: '', data_type: 'text', required: false, multi: false })
    },
    applyTemplate(tpl) {
      this.fields = tpl.fields.map(f => ({ ...f }))
    },
    async handleStartExtract() {
      this.extracting = true
      this.activeStep = 2
      try {
        const fileIds = this.uploadedFiles.filter(f => f.status === 'parsed').map(f => f.id)
        const res = await startExtract({ task_name: '合同提取_' + Date.now(), file_ids: fileIds, fields: this.fields })
        this.taskId = (res.data.data || res.data).task_id
        this.startPolling()
      } catch { this.extracting = false; this.activeStep = 1 }
    },
    startPolling() {
      this.pollTimer = setInterval(async () => {
        try {
          const res = await getExtractProgress(this.taskId)
          const d = res.data.data || res.data
          this.extractProgress = d.percent || 0
          this.extractStep = d.step || ''
          if (d.status === 'completed' || d.percent >= 100) {
            this.stopPolling()
            this.fetchResult()
          } else if (d.status === 'failed') {
            this.stopPolling()
            this.activeStep = 1
            this.$message.error('合同提取失败')
          }
        } catch { this.stopPolling() }
      }, 10000)
    },
    stopPolling() { if (this.pollTimer) { clearInterval(this.pollTimer); this.pollTimer = null } },
    async fetchResult() {
      try {
        const res = await getExtractResult(this.taskId)
        this.resultData = res.data.data || res.data
        this.resultFields = this.resultData.fields || []
        this.extracting = false
      } catch { this.extracting = false }
    },
    startEdit(row, field) { this.editingCell = row.file_id + '_' + field },
    async saveEdit(row, field) {
      this.editingCell = ''
      try { await updateExtractCell(row.id, field, row.data[field]) } catch {}
    },
    async handleExport() {
      try {
        const res = await exportExtractResult(this.taskId)
        const url = URL.createObjectURL(res.data)
        const a = document.createElement('a'); a.href = url; a.download = `提取结果.xlsx`
        document.body.appendChild(a); a.click(); document.body.removeChild(a); URL.revokeObjectURL(url)
      } catch { this.$message.error('导出失败') }
    },
    handleBack() {
      this.activeStep = 0; this.resultData = null; this.taskId = null; this.extracting = false
    },
    formatSize(bytes) {
      if (!bytes) return ''; const u = ['B','KB','MB','GB']; let i = 0; let s = bytes
      while (s >= 1024 && i < u.length-1) { s /= 1024; i++ }
      return s.toFixed(1) + u[i]
    }
  }
}
</script>

<style scoped>
.extract-page { height: 100%; display: flex; flex-direction: column; overflow: auto; }
.page-header { background: #fff; padding: 8px 20px; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; }
.page-header h2 { font-size: 16px; margin: 0; color: #333; }
.page-desc { display: none; }
.extract-steps { padding: 10px 20px; background: #fff; border-bottom: 1px solid #e4e7ed; }
.step-content { flex: 1; padding: 0; overflow: auto; }
.upload-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.upload-card >>> .el-upload-dragger { margin-bottom: 0; width: 100%; }
.upload-card >>> .el-card__body { padding: 12px 16px; }
.upload-card >>> .el-upload { width: 100%; }
.upload-card >>> .el-upload-dragger { width: 100%; padding: 16px; }
.step-actions { display: flex; gap: 8px; justify-content: center; padding: 8px 0; }
.file-list-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.file-list-card >>> .el-card__body { padding: 6px 16px; }
.file-item { display: flex; align-items: center; gap: 8px; padding: 4px 0; font-size: 13px; color: #333; }
.file-name { flex: 1; }
.file-size { color: #999; font-size: 12px; }
.field-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.field-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; flex-wrap: wrap; }
.field-empty { text-align: center; color: #999; padding: 20px; }
.template-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.template-tag { cursor: pointer; margin: 0 4px 4px 0; }
.progress-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.progress-step { margin-top: 8px; color: #409eff; font-size: 13px; }
.result-card { margin-bottom: 0; border-top: none; border-radius: 0; }
.result-toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.result-title { flex: 1; font-weight: 600; font-size: 15px; }
</style>
