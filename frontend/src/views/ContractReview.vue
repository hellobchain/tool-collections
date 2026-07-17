<template>
  <div class="contract-review">
    <div class="page-header">
      <h2>合同智能审查</h2>
      <p class="page-desc">上传合同文档，配置审查参数，自动输出审查报告</p>
    </div>

    <el-steps :active="activeStep" align-center class="review-steps">
      <el-step title="上传合同" description="支持PDF/Word" />
      <el-step title="配置审查" description="类型/立场/标准" />
      <el-step title="审查结果" description="报告与批注" />
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
          :on-change="handleFileSelect"
          :accept="'.doc,.docx,.pdf'"
          :before-upload="beforeUpload"
        >
          <i class="el-icon-upload"></i>
          <div class="el-upload__text">拖拽合同文件到此处，或<em>点击选择</em></div>
          <div slot="tip" class="el-upload__tip">支持 .doc .docx .pdf 格式，单文件不超过20MB，单次最多5份</div>
        </el-upload>
      </el-card>

      <el-card v-if="uploadedFiles.length" class="file-list-card">
        <div class="file-list-header">
          <span>已上传文件（{{ uploadedFiles.length }}份）</span>
        </div>
        <div v-for="file in uploadedFiles" :key="file.id" class="file-item">
          <div class="file-info">
            <i class="el-icon-document"></i>
            <span class="file-name">{{ file.name }}</span>
            <span class="file-size">{{ file.size }}</span>
          </div>
          <div class="file-status">
            <el-tag v-if="file.status==='parsed'" type="success" size="mini" effect="dark">解析完成</el-tag>
            <el-tag v-else-if="file.status==='parsing'" type="warning" size="mini">解析中...</el-tag>
            <el-tag v-else-if="file.status==='uploading'" type="primary" size="mini">上传中...</el-tag>
            <el-tag v-else-if="file.status==='failed'" type="danger" size="mini">解析失败</el-tag>
            <el-button v-if="file.status==='parsed'" type="text" size="mini" icon="el-icon-view" @click="previewFile(file)">查看</el-button>
            <el-button type="text" size="mini" icon="el-icon-delete" @click="removeFile(file)" style="color:#999" />
          </div>
        </div>
      </el-card>

      <div class="step-actions">
        <el-button :disabled="!canProceedStep1" type="primary" @click="activeStep=1">下一步</el-button>
      </div>
    </div>

    <!-- Step 2: Configure -->
    <div v-show="activeStep === 1" class="step-content">
      <el-card class="config-card">
        <el-form label-width="120px">
          <el-form-item label="合同类型">
            <el-cascader
              v-model="configType"
              :options="contractTypes"
              :props="{ expandTrigger: 'hover', label: 'label', value: 'value' }"
              placeholder="选择合同类型"
              clearable
              style="width:360px"
            />
            <el-input v-if="configType && configType[0]==='other'" v-model="customType" placeholder="请输入自定义类型" style="width:240px;margin-left:8px" size="small" />
          </el-form-item>

          <el-form-item label="审查立场">
            <el-radio-group v-model="selectedPosition">
              <div v-for="p in positions" :key="p.value" class="position-item">
                <el-radio :label="p.value" border>{{ p.label }}</el-radio>
                <div class="position-desc">
                  <span class="position-focus">{{ p.focus }}</span>
                </div>
              </div>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="审查标准">
            <el-checkbox-group v-model="selectedStandards">
              <div v-for="s in standards" :key="s.value" class="standard-item">
                <el-checkbox :label="s.value" border>{{ s.label }}</el-checkbox>
                <div class="standard-desc">{{ s.desc }}</div>
              </div>
            </el-checkbox-group>
          </el-form-item>
        </el-form>
      </el-card>

      <el-card class="summary-card">
        <div slot="header">配置摘要</div>
        <div class="summary-body">
          <el-row :gutter="16">
            <el-col :span="8"><div class="summary-item"><label>合同类型</label><span>{{ configTypeLabel || '未选择' }}</span></div></el-col>
            <el-col :span="8"><div class="summary-item"><label>审查立场</label><span>{{ positionLabel || '未选择' }}</span></div></el-col>
            <el-col :span="8"><div class="summary-item"><label>审查标准</label><span>{{ standardsLabel || '未选择' }}</span></div></el-col>
          </el-row>
        </div>
      </el-card>

      <div class="step-actions">
        <el-button @click="activeStep=0">上一步</el-button>
        <el-button :disabled="!canProceedStep2" type="primary" :loading="reviewing" @click="handleStartReview">
          {{ reviewing ? '审查中...' : '开始审查' }}
        </el-button>
      </div>
    </div>

    <!-- Step 3: Results -->
    <div v-show="activeStep === 2" class="step-content report-step">
      <div class="report-layout">
        <div class="report-main">
          <!-- Overview -->
          <el-card class="overview-card" v-if="report">
            <div class="overview-header">
              <span class="overview-title">审查报告：{{ report.file_name || report.name || '' }}</span>
              <div class="overview-actions">
                <el-button size="mini" icon="el-icon-download" @click="handleExport('word')">Word</el-button>
                <el-button size="mini" icon="el-icon-download" @click="handleExport('pdf')">PDF</el-button>
                <el-button size="mini" icon="el-icon-download" @click="handleExport('excel')">Excel</el-button>
              </div>
            </div>
            <div class="overview-body">
              <div class="overview-config">
                <span>立场：{{ positionLabel }}</span>
                <span>标准：{{ standardsLabel }}</span>
                <span>规则数：{{ report.total_rules || (report.items||[]).length }} 项</span>
              </div>
              <div class="risk-stat">
                <div class="risk-item high"><span class="risk-num">{{ riskCounts.high }}</span>高风险</div>
                <div class="risk-item medium"><span class="risk-num">{{ riskCounts.medium }}</span>中风险</div>
                <div class="risk-item low"><span class="risk-num">{{ riskCounts.low }}</span>低风险</div>
                <div class="risk-item pass"><span class="risk-num">{{ riskCounts.pass }}</span>通过</div>
              </div>
              <div class="overview-conclusion" v-if="report.conclusion">
                <el-tag :type="conclusionTagType" size="medium">{{ report.conclusion }}</el-tag>
              </div>
            </div>
          </el-card>

          <!-- Progress -->
          <el-card v-if="reviewing && !report" class="progress-card">
            <div class="progress-info">
              <span>审查进度：{{ reviewProgress ? reviewProgress.percent || 0 : 0 }}%</span>
              <span v-if="reviewProgress && reviewProgress.current_rule" class="current-rule">当前：{{ reviewProgress.current_rule }}</span>
            </div>
            <el-progress :percentage="reviewProgress ? reviewProgress.percent || 0 : 0" :status="reviewProgress && reviewProgress.percent >= 100 ? 'success' : ''" />
            <div v-if="reviewProgress" class="progress-risk">
              <span>已发现：<em style="color:#f56c6c">高 {{ reviewProgress.high_risk || 0 }}</em> / <em style="color:#e6a23c">中 {{ reviewProgress.medium_risk || 0 }}</em> / <em style="color:#409eff">低 {{ reviewProgress.low_risk || 0 }}</em></span>
            </div>
          </el-card>

          <!-- Filters -->
          <el-card v-if="report" class="filter-card">
            <div class="filter-bar">
              <el-radio-group v-model="filterLevel" size="mini">
                <el-radio-button label="">全部</el-radio-button>
                <el-radio-button label="high">高风险</el-radio-button>
                <el-radio-button label="medium">中风险</el-radio-button>
                <el-radio-button label="low">低风险</el-radio-button>
              </el-radio-group>
              <el-input v-model="searchKeyword" placeholder="搜索关键词..." size="mini" style="width:200px" prefix-icon="el-icon-search" clearable />
            </div>
          </el-card>

          <!-- Risk list -->
          <div v-if="report" class="risk-list">
            <div v-for="item in filteredItems" :key="item.id" class="risk-item-card" :class="'level-' + item.level">
              <div class="risk-item-header">
                <el-tag :type="levelTagType(item.level)" size="mini" effect="dark">{{ riskLevelMap[item.level]?.label || item.level }}</el-tag>
                <span class="risk-section" v-if="item.section">第{{ item.section }}条</span>
                <span class="risk-rule-name">{{ item.rule_name || item.name }}</span>
                <div class="risk-item-actions">
                  <el-button v-if="item.status!=='resolved'" type="text" size="mini" @click="handleMarkResolved(item)">标记已处理</el-button>
                  <el-button v-if="item.status!=='ignored'" type="text" size="mini" @click="handleMarkIgnored(item)">忽略</el-button>
                  <el-button type="text" size="mini" @click="handleAddComment(item)">批注</el-button>
                </div>
              </div>
              <div class="risk-item-body">
                <div class="risk-field"><label>问题：</label><span>{{ item.description || item.issue }}</span></div>
                <div class="risk-field" v-if="item.suggestion"><label>建议：</label><span>{{ item.suggestion }}</span></div>
                <div class="risk-field" v-if="item.law_ref"><label>依据：</label><span>{{ item.law_ref }}</span></div>
                <div class="risk-field" v-if="item.original_text">
                  <label>原文：</label>
                  <span class="original-text" @click="locateText(item.original_text)">{{ item.original_text }}</span>
                </div>
              </div>
              <div v-if="item.comment" class="risk-item-comment">
                <i class="el-icon-edit-outline"></i> {{ item.comment }}
              </div>
            </div>
            <el-empty v-if="!filteredItems.length" description="暂无审查意见" />
          </div>
        </div>

        <!-- Right: Contract text -->
        <div v-if="contractText" class="contract-text-panel">
          <div class="panel-header">合同原文</div>
          <div ref="textContent" class="panel-content" v-html="highlightedText"></div>
        </div>
      </div>

      <div class="step-actions" v-if="activeStep===2">
        <el-button @click="handleBackToUpload">返回工作台</el-button>
        <el-button type="primary" @click="$router.push('/contract-history')">查看历史记录</el-button>
      </div>
    </div>
  </div>
</template>

<script>
import { mapState, mapGetters } from 'vuex'
import { uploadContract } from '@/api/contract'

export default {
  name: 'ContractReview',
  data() {
    return {
      activeStep: 0,
      configType: [],
      customType: '',
      pollTimer: null,
      filterLevel: '',
      searchKeyword: '',
      highlightText: ''
    }
  },
  computed: {
    ...mapState('contract', ['uploadedFiles', 'contractTypes', 'positions', 'standards', 'selectedPosition', 'selectedStandards', 'reviewing', 'reviewProgress', 'report', 'contractText', 'riskLevelMap']),
    canProceedStep1() {
      return this.uploadedFiles.some(f => f.status === 'parsed')
    },
    canProceedStep2() {
      return this.configType && this.configType.length && this.selectedPosition && this.selectedStandards.length
    },
    configTypeLabel() {
      if (!this.configType || !this.configType.length) return ''
      const cat = this.contractTypes.find(c => c.value === this.configType[0])
      if (!cat) return this.configType[0]
      if (this.configType.length === 1) return cat.label
      const sub = cat.children.find(c => c.value === this.configType[1])
      return sub ? `${cat.label} - ${sub.label}` : cat.label
    },
    positionLabel() {
      const p = this.positions.find(p => p.value === this.selectedPosition)
      return p ? p.label : ''
    },
    standardsLabel() {
      return this.selectedStandards.map(v => { const s = this.standards.find(st => st.value === v); return s ? s.label : v }).join('、') || ''
    },
    riskCounts() {
      if (!this.report) return { high: 0, medium: 0, low: 0, pass: 0 }
      const items = this.report.items || []
      return {
        high: items.filter(i => i.level === 'high').length,
        medium: items.filter(i => i.level === 'medium').length,
        low: items.filter(i => i.level === 'low').length,
        pass: items.filter(i => i.level === 'pass' || !i.level).length
      }
    },
    conclusionTagType() {
      const c = this.report?.conclusion || ''
      if (c.includes('不通过')) return 'danger'
      if (c.includes('有条件')) return 'warning'
      return 'success'
    },
    filteredItems() {
      if (!this.report) return []
      const items = this.report.items || []
      let result = this.filterLevel ? items.filter(i => i.level === this.filterLevel) : items
      if (this.searchKeyword) {
        const kw = this.searchKeyword.toLowerCase()
        result = result.filter(i =>
          (i.description || '').toLowerCase().includes(kw) ||
          (i.rule_name || '').toLowerCase().includes(kw) ||
          (i.section || '').toLowerCase().includes(kw)
        )
      }
      const order = { high: 0, medium: 1, low: 2, pass: 3 }
      return [...result].sort((a, b) => (order[a.level] ?? 9) - (order[b.level] ?? 9))
    },
    highlightedText() {
      if (!this.contractText) return ''
      if (!this.highlightText) return this.escapeHtml(this.contractText)
      const escaped = this.escapeHtml(this.contractText)
      const kw = this.escapeHtml(this.highlightText)
      return escaped.replace(new RegExp(kw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi'), m => `<mark class="highlight">${m}</mark>`)
    }
  },
  watch: {
    activeStep(val) {
      if (val === 2) {
        this.$store.commit('contract/SET_SELECTED_TYPE', this.configType[0] || '')
        this.$store.commit('contract/SET_SELECTED_SUB_TYPE', this.configType[1] || '')
        if (this.configType[0] === 'other') {
          this.$store.commit('contract/SET_CUSTOM_TYPE', this.customType)
        }
      }
    }
  },
  mounted() {
    this.$store.commit('contract/SET_SELECTED_POSITION', 'party_a')
  },
  beforeDestroy() {
    this.stopPolling()
  },
  methods: {
    beforeUpload(file) {
      const isDoc = /\.(doc|docx|pdf)$/i.test(file.name)
      if (!isDoc) { this.$message.error('仅支持 .doc .docx .pdf 格式'); return false }
      if (file.size > 20 * 1024 * 1024) { this.$message.error('文件大小不能超过 20MB'); return false }
      if (this.uploadedFiles.length >= 5) { this.$message.error('单次最多上传5份文件'); return false }
      return true
    },
    async handleFileSelect(file) {
      if (!file || !file.raw) return
      if (!this.beforeUpload(file.raw)) { this.$refs.upload.clearFiles(); return }
      const rawFile = file.raw
      const tempId = '_temp_' + Date.now()
      this.$store.commit('contract/ADD_UPLOADED_FILE', {
        id: tempId,
        name: rawFile.name,
        size: this.formatSize(rawFile.size),
        status: 'uploading',
        rawFile
      })
      try {
        const res = await uploadContract(rawFile, p => {
          const pct = Math.round((p.loaded / p.total) * 100)
          this.$store.commit('contract/UPDATE_FILE_STATUS', { id: tempId, status: pct < 100 ? 'uploading' : 'parsing' })
        })
        const serverFile = res.data.data || res.data
        this.$store.commit('contract/REMOVE_UPLOADED_FILE', tempId)
        this.$store.commit('contract/ADD_UPLOADED_FILE', {
          id: serverFile.id,
          name: serverFile.name || rawFile.name,
          size: serverFile.size || this.formatSize(rawFile.size),
          status: 'parsed'
        })
        this.$message.success(`${rawFile.name} 上传解析完成`)
      } catch (e) {
        this.$store.commit('contract/UPDATE_FILE_STATUS', { id: tempId, status: 'failed', msg: e.message || '上传失败' })
        this.$message.error(`${rawFile.name} 上传失败`)
      }
      this.$refs.upload.clearFiles()
    },
    removeFile(file) {
      this.$confirm('确定移除此文件？', '提示', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }).then(() => {
        this.$store.commit('contract/REMOVE_UPLOADED_FILE', file.id)
      }).catch(() => {})
    },
    previewFile(file) {
      this.$store.dispatch('contract/fetchContractText', file.id)
    },
    async handleStartReview() {
      try {
        const { taskId } = await this.$store.dispatch('contract/startReview')
        this.activeStep = 2
        this.startPolling()
      } catch (e) {
        this.$message.error('启动审查失败：' + (e.message || '未知错误'))
      }
    },
    startPolling() {
      this.pollTimer = setInterval(async () => {
        try {
          const progress = await this.$store.dispatch('contract/pollProgress')
          if (progress && progress.percent >= 100) {
            this.stopPolling()
            await this.$store.dispatch('contract/fetchReport')
            this.$store.commit('contract/SET_REVIEWING', false)
            if (this.uploadedFiles.length) {
              this.$store.dispatch('contract/fetchContractText', this.uploadedFiles[0].id)
            }
          }
        } catch (e) {
          this.stopPolling()
          this.$store.commit('contract/SET_REVIEWING', false)
        }
      }, 2000)
    },
    stopPolling() {
      if (this.pollTimer) { clearInterval(this.pollTimer); this.pollTimer = null }
    },
    locateText(text) {
      this.highlightText = text
      this.$nextTick(() => {
        const el = this.$refs.textContent
        if (!el) return
        const mark = el.querySelector('mark.highlight')
        if (mark) mark.scrollIntoView({ behavior: 'smooth', block: 'center' })
      })
    },
    async handleMarkResolved(item) {
      const reportId = this.report?.id
      if (!reportId) return
      await this.$store.dispatch('contract/updateItem', { reportId, itemId: item.id, payload: { status: 'resolved' } })
      this.$message.success('已标记为已处理')
    },
    async handleMarkIgnored(item) {
      const reportId = this.report?.id
      if (!reportId) return
      await this.$store.dispatch('contract/updateItem', { reportId, itemId: item.id, payload: { status: 'ignored' } })
      this.$message.success('已忽略')
    },
    handleAddComment(item) {
      this.$prompt('请输入批注内容', '添加批注', { inputType: 'textarea', inputValue: item.comment || '' }).then(async ({ value }) => {
        const reportId = this.report?.id
        if (!reportId) return
        await this.$store.dispatch('contract/updateItem', { reportId, itemId: item.id, payload: { comment: value } })
        this.$message.success('批注已保存')
      }).catch(() => {})
    },
    handleExport(format) {
      const reportId = this.report?.id
      if (!reportId) return
      this.$store.dispatch('contract/exportReport', { reportId, format }).then(blob => {
        const extMap = { word: 'docx', pdf: 'pdf', excel: 'xlsx' }
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `审查报告_${this.report.file_name || 'report'}.${extMap[format] || format}`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)
      }).catch(() => this.$message.error('导出失败'))
    },
    handleBackToUpload() {
      this.$store.commit('contract/RESET_REVIEW')
      this.uploadedFiles.splice(0)
      this.activeStep = 0
      this.filterLevel = ''
      this.searchKeyword = ''
      this.highlightText = ''
    },
    levelTagType(level) {
      const map = { high: 'danger', medium: 'warning', low: 'primary', pass: 'success' }
      return map[level] || 'info'
    },
    formatSize(bytes) {
      if (!bytes) return ''
      const units = ['B', 'KB', 'MB', 'GB']
      let i = 0
      let size = bytes
      while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
      return size.toFixed(1) + units[i]
    },
    escapeHtml(text) {
      const div = document.createElement('div')
      div.textContent = text
      return div.innerHTML
    }
  }
}
</script>

<style scoped>
.contract-review {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: auto;
}
.page-header {
  background: #fff;
  padding: 16px 32px;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
}
.page-header h2 { font-size: 20px; margin: 0 0 4px; color: #333; }
.page-desc { color: #999; font-size: 14px; margin: 0; }
.review-steps { padding: 24px 32px; background: #fff; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; }
.step-content { flex: 1; padding: 16px 32px; overflow: auto; }
.upload-card { margin-bottom: 12px; }
.upload-card >>> .el-upload-dragger { margin-bottom: 0; }
.file-list-card { margin-bottom: 12px; }
.file-list-header { font-size: 14px; color: #333; margin-bottom: 12px; }
.file-item { display: flex; align-items: center; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #f0f0f0; }
.file-item:last-child { border-bottom: none; }
.file-info { display: flex; align-items: center; gap: 8px; }
.file-info .el-icon-document { color: #409eff; font-size: 18px; }
.file-name { color: #333; }
.file-size { color: #999; font-size: 12px; }
.file-status { display: flex; align-items: center; gap: 8px; }
.config-card { margin-bottom: 12px; }
.position-item, .standard-item { margin-bottom: 8px; }
.position-desc, .standard-desc { font-size: 12px; color: #999; margin: 4px 0 4px 24px; }
.summary-card { margin-bottom: 12px; }
.summary-item label { display: block; font-size: 12px; color: #999; margin-bottom: 4px; }
.step-actions { display: flex; gap: 12px; justify-content: center; padding: 16px 0; }
.report-layout { display: flex; gap: 16px; }
.report-main { flex: 1; min-width: 0; }
.contract-text-panel { width: 400px; flex-shrink: 0; background: #fff; border: 1px solid #e4e7ed; border-radius: 4px; display: flex; flex-direction: column; max-height: calc(100vh - 300px); }
.panel-header { padding: 12px 16px; font-weight: 600; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; }
.panel-content { padding: 16px; overflow: auto; flex: 1; font-size: 13px; line-height: 1.8; white-space: pre-wrap; word-break: break-all; }
.panel-content >>> mark.highlight { background: #ffd666; padding: 0 2px; }
.overview-card { margin-bottom: 12px; }
.overview-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.overview-title { font-size: 16px; font-weight: 600; color: #333; }
.overview-actions { display: flex; gap: 8px; }
.overview-config { display: flex; gap: 16px; font-size: 13px; color: #666; margin-bottom: 12px; }
.risk-stat { display: flex; gap: 24px; margin-bottom: 12px; }
.risk-item { display: flex; align-items: center; gap: 6px; font-size: 14px; }
.risk-num { font-size: 28px; font-weight: 700; }
.risk-item.high .risk-num { color: #f56c6c; }
.risk-item.medium .risk-num { color: #e6a23c; }
.risk-item.low .risk-num { color: #409eff; }
.risk-item.pass .risk-num { color: #67c23a; }
.progress-card { margin-bottom: 12px; }
.progress-info { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 13px; color: #666; }
.current-rule { color: #409eff; }
.progress-risk { margin-top: 8px; font-size: 13px; color: #666; }
.filter-card { margin-bottom: 12px; }
.filter-bar { display: flex; align-items: center; gap: 12px; }
.risk-list { display: flex; flex-direction: column; gap: 8px; }
.risk-item-card { background: #fff; border: 1px solid #e4e7ed; border-radius: 4px; padding: 12px 16px; border-left: 4px solid #e4e7ed; }
.risk-item-card.level-high { border-left-color: #f56c6c; }
.risk-item-card.level-medium { border-left-color: #e6a23c; }
.risk-item-card.level-low { border-left-color: #409eff; }
.risk-item-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.risk-section { color: #666; font-size: 13px; }
.risk-rule-name { flex: 1; font-weight: 600; color: #333; }
.risk-item-actions { display: flex; gap: 4px; }
.risk-item-body { margin-bottom: 4px; }
.risk-field { margin: 4px 0; font-size: 13px; line-height: 1.6; }
.risk-field label { color: #999; }
.original-text { color: #606266; cursor: pointer; border-bottom: 1px dashed #dcdfe6; }
.original-text:hover { color: #409eff; }
.risk-item-comment { margin-top: 8px; padding: 6px 10px; background: #fdf6ec; border-radius: 4px; font-size: 13px; color: #e6a23c; }
@media (max-width: 1200px) {
  .contract-text-panel { display: none; }
  .report-layout { flex-direction: column; }
}
</style>
