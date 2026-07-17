import * as contractApi from '@/api/contract'

const CONTRACT_TYPES = [
  {
    label: '买卖合同', value: 'purchase',
    children: [
      { label: '设备采购', value: 'equipment' },
      { label: '原材料采购', value: 'raw_material' },
      { label: '服务采购', value: 'service_procurement' },
      { label: '框架协议', value: 'framework' }
    ]
  },
  {
    label: '租赁合同', value: 'lease',
    children: [
      { label: '房屋租赁', value: 'housing' },
      { label: '设备租赁', value: 'equipment_lease' },
      { label: '场地租赁', value: 'venue' }
    ]
  },
  {
    label: '服务合同', value: 'service',
    children: [
      { label: '技术服务', value: 'tech_service' },
      { label: '咨询服务', value: 'consulting' },
      { label: '物业服务', value: 'property' },
      { label: '运输服务', value: 'transport' }
    ]
  },
  {
    label: '劳动合同', value: 'labor',
    children: [
      { label: '劳动合同', value: 'employment' },
      { label: '劳务派遣', value: 'dispatch' },
      { label: '保密协议', value: 'nda' },
      { label: '竞业限制', value: 'non_compete' }
    ]
  },
  {
    label: '投融资合同', value: 'investment',
    children: [
      { label: '股权转让', value: 'equity_transfer' },
      { label: '增资协议', value: 'capital_increase' },
      { label: '借款合同', value: 'loan' },
      { label: '担保合同', value: 'guarantee' }
    ]
  },
  {
    label: '工程合同', value: 'engineering',
    children: [
      { label: '施工总包', value: 'construction' },
      { label: '分包合同', value: 'subcontract' },
      { label: '勘察设计', value: 'survey_design' },
      { label: '监理合同', value: 'supervision' }
    ]
  },
  {
    label: '知识产权合同', value: 'ip',
    children: [
      { label: '专利许可', value: 'patent_license' },
      { label: '商标转让', value: 'trademark' },
      { label: '版权授权', value: 'copyright' },
      { label: '技术开发', value: 'tech_dev' }
    ]
  },
  { label: '其他', value: 'other', children: [] }
]

const POSITIONS = [
  { label: '甲方立场', value: 'party_a', desc: '作为合同甲方（采购方/委托方/出租方等）', focus: '重点审查履约标准、验收条款、付款节奏、违约责任对等性、质保期限' },
  { label: '乙方立场', value: 'party_b', desc: '作为合同乙方（供应商/服务方/承租方等）', focus: '重点审查付款节点、验收条件合理性、违约责任上限、知识产权归属' },
  { label: '中立立场', value: 'neutral', desc: '作为合同见证/审批方', focus: '全面审查条款合规性、公平性，关注法律风险' }
]

const STANDARDS = [
  { label: '内部合规标准', value: 'internal', desc: '公司内部管理制度、审批权限、合同模板规范等' },
  { label: '法律法规标准', value: 'legal', desc: '民法典合同编、相关行业法规、司法解释等' },
  { label: '行业标准', value: 'industry', desc: '根据合同类型自动关联行业标准' },
  { label: '自定义标准', value: 'custom', desc: '用户可选择已配置的自定义审查清单' }
]

const RISK_LEVEL_MAP = { high: { label: '高风险', color: '#f56c6c', icon: 'el-icon-warning' }, medium: { label: '中风险', color: '#e6a23c', icon: 'el-icon-warning' }, low: { label: '低风险', color: '#409eff', icon: 'el-icon-info' }, pass: { label: '通过', color: '#67c23a', icon: 'el-icon-success' } }

export default {
  namespaced: true,
  state: {
    contractTypes: CONTRACT_TYPES,
    positions: POSITIONS,
    standards: STANDARDS,
    riskLevelMap: RISK_LEVEL_MAP,

    uploadedFiles: [],
    selectedType: '',
    selectedSubType: '',
    selectedPosition: '',
    selectedStandards: [],
    customType: '',

    reviewing: false,
    reviewTaskId: null,
    reviewProgress: null,
    report: null,
    contractText: '',

    historyList: [],
    historyTotal: 0,
    historyLoading: false
  },
  mutations: {
    SET_UPLOADED_FILES(state, files) { state.uploadedFiles = files },
    ADD_UPLOADED_FILE(state, file) { state.uploadedFiles.push(file) },
    UPDATE_FILE_STATUS(state, { id, status, msg }) {
      const f = state.uploadedFiles.find(f => f.id === id)
      if (f) { f.status = status; if (msg) f.errorMsg = msg }
    },
    REMOVE_UPLOADED_FILE(state, id) { state.uploadedFiles = state.uploadedFiles.filter(f => f.id !== id) },
    SET_SELECTED_TYPE(state, val) { state.selectedType = val },
    SET_SELECTED_SUB_TYPE(state, val) { state.selectedSubType = val },
    SET_SELECTED_POSITION(state, val) { state.selectedPosition = val },
    SET_SELECTED_STANDARDS(state, val) { state.selectedStandards = val },
    SET_CUSTOM_TYPE(state, val) { state.customType = val },
    SET_REVIEWING(state, val) { state.reviewing = val },
    SET_REVIEW_TASK_ID(state, val) { state.reviewTaskId = val },
    SET_REVIEW_PROGRESS(state, val) { state.reviewProgress = val },
    SET_REPORT(state, val) { state.report = val },
    SET_CONTRACT_TEXT(state, val) { state.contractText = val },
    SET_HISTORY_LIST(state, val) { state.historyList = val },
    SET_HISTORY_TOTAL(state, val) { state.historyTotal = val },
    SET_HISTORY_LOADING(state, val) { state.historyLoading = val },
    UPDATE_REPORT_ITEM(state, { itemId, payload }) {
      if (!state.report) return
      const items = state.report.items || []
      const idx = items.findIndex(i => i.id === itemId)
      if (idx >= 0) Object.assign(items[idx], payload)
    },
    RESET_REVIEW(state) {
      state.reviewing = false
      state.reviewTaskId = null
      state.reviewProgress = null
      state.report = null
      state.contractText = ''
    }
  },
  actions: {
    async uploadFile({ commit }, { file, onProgress }) {
      const res = await contractApi.uploadContract(file, onProgress)
      const { data } = res.data
      commit('ADD_UPLOADED_FILE', { ...data, file_uuid: data.file_uuid || '', status: 'parsed', progress: 100 })
      return data
    },
    async deleteFile({ commit }, fileId) {
      await contractApi.deleteContract(fileId)
      commit('REMOVE_UPLOADED_FILE', fileId)
    },
    async startReview({ state, commit }) {
      commit('SET_REVIEWING', true)
      commit('SET_REVIEW_PROGRESS', null)
      commit('SET_REPORT', null)
      try {
        const params = {
          file_ids: state.uploadedFiles.filter(f => f.status === 'parsed').map(f => f.id),
          contract_type: state.selectedSubType || state.selectedType,
          position: state.selectedPosition,
          standards: state.selectedStandards,
          custom_type: state.customType || undefined
        }
        const res = await contractApi.startReview(params)
        const { task_id, report_id } = res.data.data || res.data
        commit('SET_REVIEW_TASK_ID', task_id || report_id)
        return { taskId: task_id, reportId: report_id }
      } catch (e) {
        commit('SET_REVIEWING', false)
        throw e
      }
    },
    async pollProgress({ state, commit }) {
      if (!state.reviewTaskId) return
      const res = await contractApi.getReviewProgress(state.reviewTaskId)
      const progress = res.data.data || res.data
      commit('SET_REVIEW_PROGRESS', progress)
      return progress
    },
    async fetchReport({ state, commit }) {
      if (!state.reviewTaskId) return
      const res = await contractApi.getReviewReport(state.reviewTaskId)
      const report = res.data.data || res.data
      commit('SET_REPORT', report)
      return report
    },
    async fetchContractText({ commit }, fileId) {
      const res = await contractApi.getContractText(fileId)
      const text = res.data.data || (typeof res.data === 'string' ? res.data : '')
      commit('SET_CONTRACT_TEXT', text)
    },
    async updateItem({ commit }, { reportId, itemId, payload }) {
      await contractApi.updateReviewItem(reportId, itemId, payload)
      commit('UPDATE_REPORT_ITEM', { itemId, payload })
    },
    async fetchHistory({ commit }, params) {
      commit('SET_HISTORY_LOADING', true)
      try {
        const res = await contractApi.getHistory(params)
        const { list, total } = res.data.data || res.data
        commit('SET_HISTORY_LIST', list || [])
        commit('SET_HISTORY_TOTAL', total || 0)
      } finally {
        commit('SET_HISTORY_LOADING', false)
      }
    },
    async deleteHistory({ commit }, reportId) {
      await contractApi.deleteHistory(reportId)
    },
    async exportReport(_, { reportId, format }) {
      const res = await contractApi.exportReport(reportId, format)
      return res.data
    }
  }
}
