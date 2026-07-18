<template>
  <div class="contract-history">
    <div class="page-header">
      <h2>📂 历史审查记录</h2>
    </div>

    <el-card class="filter-card">
      <div class="filter-bar">
        <el-date-picker v-model="dateRange" type="daterange" range-separator="~" start-placeholder="开始日期" end-placeholder="结束日期" size="small" value-format="yyyy-MM-dd" style="width:260px" />
        <el-select v-model="filterType" placeholder="合同类型" size="small" clearable style="width:150px">
          <el-option v-for="t in flatTypes" :key="t.value" :label="t.label" :value="t.value" />
        </el-select>
        <el-input v-model="searchText" placeholder="搜索合同名称..." size="small" style="width:200px" prefix-icon="el-icon-search" clearable />
        <el-button type="primary" size="small" icon="el-icon-search" @click="handleSearch">搜索</el-button>
      </div>
    </el-card>

    <el-card class="list-card">
      <el-table :data="historyList" v-loading="historyLoading" stripe style="width:100%">
        <el-table-column prop="file_name" label="合同名称" min-width="200">
          <template slot-scope="{ row }">
            <el-button type="text" @click="viewReport(row)">{{ row.file_name || row.name }}</el-button>
          </template>
        </el-table-column>
        <el-table-column prop="contract_type_label" label="类型" width="100" />
        <el-table-column prop="reviewer" label="审查人" width="100" />
        <el-table-column prop="review_start_time" label="审查开始时间" width="160" />
        <el-table-column prop="review_end_time" label="审查结束时间" width="160" />
        <el-table-column label="审查结果" width="100">
          <template slot-scope="{ row }">
            <el-tag v-if="row.status==='completed'" type="success" size="mini">已完成</el-tag>
            <el-tag v-else-if="row.status==='failed'" type="danger" size="mini">失败</el-tag>
            <el-tag v-else-if="row.status==='running'" type="warning" size="mini">审查中</el-tag>
            <el-tag v-else-if="row.status==='pending'" type="info" size="mini">待审查</el-tag>
            <span v-else>{{ row.status }}</span>
          </template>
        </el-table-column>
        <el-table-column label="审查进度" width="150">
          <template slot-scope="{ row }">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: (row.progress || 0) + '%' }"></div>
              <span class="progress-text">{{ row.progress || 0 }}%</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="风险统计" width="180">
          <template slot-scope="{ row }">
            <span v-if="row.risk_stats">
              <span style="color:#f56c6c">{{ row.risk_stats.high || 0 }}高</span>
              / <span style="color:#e6a23c">{{ row.risk_stats.medium || 0 }}中</span>
              / <span style="color:#409eff">{{ row.risk_stats.low || 0 }}低</span>
            </span>
            <span v-else>{{ row.total_risks || 0 }}项</span>
          </template>
        </el-table-column>
        <el-table-column label="综合评级" width="120">
          <template slot-scope="{ row }">
            <el-tag :type="conclusionType(row.conclusion)" size="mini">{{ row.conclusion || '-' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="170" fixed="right">
          <template slot-scope="{ row }">
            <el-button type="text" size="mini" icon="el-icon-view" @click="viewReport(row)">查看</el-button>
            <el-dropdown size="mini" trigger="click" @command="cmd => handleExport(row, cmd)">
              <el-button type="text" size="mini" icon="el-icon-download">导出</el-button>
              <el-dropdown-menu slot="dropdown">
                <el-dropdown-item command="word">Word</el-dropdown-item>
                <el-dropdown-item command="excel">Excel</el-dropdown-item>
              </el-dropdown-menu>
            </el-dropdown>
            <el-button type="text" size="mini" icon="el-icon-delete" style="color:#999" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap" v-if="historyTotal > 0">
        <el-pagination
          @current-change="handlePageChange"
          :current-page="page"
          :page-size="pageSize"
          :total="historyTotal"
          layout="total, prev, pager, next"
          small
        />
      </div>
    </el-card>

    <!-- Detail dialog -->
    <el-dialog :visible.sync="detailVisible" fullscreen title="审查报告详情" class="report-dialog">
      <div class="dialog-report" v-if="detailReport">
        <div class="dialog-overview">
          <h3>审查报告：{{ detailReport.file_name || detailReport.name }}</h3>
          <div class="dialog-meta">
            <span>合同类型：{{ detailReport.contract_type_label || detailReport.contract_type }}</span>
            <span>审查开始时间：{{ detailReport.review_start_time }}</span>
            <span>审查结束时间：{{ detailReport.review_end_time }}</span>
            <span>审查人：{{ detailReport.reviewer }}</span>
          </div>
          <div class="risk-stat" v-if="detailReport.risk_stats">
            <div class="risk-item high"><span class="risk-num">{{ detailReport.risk_stats.high || 0 }}</span>高风险</div>
            <div class="risk-item medium"><span class="risk-num">{{ detailReport.risk_stats.medium || 0 }}</span>中风险</div>
            <div class="risk-item low"><span class="risk-num">{{ detailReport.risk_stats.low || 0 }}</span>低风险</div>
          </div>
          <div v-if="detailReport.conclusion" class="dialog-conclusion">
            <el-tag :type="conclusionType(detailReport.conclusion)" size="medium">{{ detailReport.conclusion }}</el-tag>
          </div>
        </div>

        <el-table :data="detailReport.items || []" stripe size="small" style="width:100%">
          <el-table-column label="风险等级" width="80">
            <template slot-scope="{ row }">
              <el-tag :type="levelType(row.level)" size="mini" effect="dark">{{ riskLevelMap[row.level]?.label || row.level }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="section" label="条款" width="80" />
          <el-table-column prop="rule_name" label="规则名称" min-width="140" />
          <el-table-column prop="description" label="问题描述" min-width="200" show-overflow-tooltip />
          <el-table-column prop="suggestion" label="修改建议" min-width="200" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="80">
            <template slot-scope="{ row }">
              <span v-if="row.status==='resolved'" style="color:#67c23a">已处理</span>
              <span v-else-if="row.status==='ignored'" style="color:#999">已忽略</span>
              <span v-else-if="row.status==='open'" style="color:green">公开</span>
              <span v-else>-</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <div class="dialog-actions">
        <el-button @click="detailVisible=false">关闭</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { mapState } from 'vuex'

export default {
  name: 'ContractHistory',
  data() {
    return {
      page: 1,
      pageSize: 15,
      dateRange: null,
      filterType: '',
      searchText: '',
      detailVisible: false,
      detailReport: null,
      flatTypes: []
    }
  },
  computed: {
    ...mapState('contract', ['historyList', 'historyTotal', 'historyLoading', 'riskLevelMap', 'contractTypes'])
  },
  created() {
    this.flattenTypes()
    this.fetchList()
  },
  methods: {
    flattenTypes() {
      this.flatTypes = []
      this.contractTypes.forEach(cat => {
        this.flatTypes.push({ label: cat.label, value: cat.value })
        if (cat.children) {
          cat.children.forEach(sub => {
            this.flatTypes.push({ label: `${cat.label} - ${sub.label}`, value: sub.value })
          })
        }
      })
    },
    fetchList() {
      const params = { page: this.page, page_size: this.pageSize }
      if (this.filterType) params.contract_type = this.filterType
      if (this.searchText) params.keyword = this.searchText
      if (this.dateRange && this.dateRange.length === 2) {
        params.start_date = this.dateRange[0]
        params.end_date = this.dateRange[1]
      }
      this.$store.dispatch('contract/fetchHistory', params)
    },
    handleSearch() {
      this.page = 1
      this.fetchList()
    },
    handlePageChange(val) {
      this.page = val
      this.fetchList()
    },
    async viewReport(row) {
      try {
        const res = await import('@/api/contract').then(m => m.getReviewReport(row.id || row.report_id))
        this.detailReport = res.data.data || res.data
        this.detailVisible = true
      } catch {
        this.$message.error('获取报告详情失败')
      }
    },
    async handleExport(row, format) {
      const { exportReport } = await import('@/api/contract')
      try {
        const res = await exportReport(row.id || row.report_id, format)
        const extMap = { word: 'docx', excel: 'xlsx' }
        const blob = res.data
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `审查报告_${row.file_name || 'report'}.${extMap[format] || format}`
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(url)
      } catch {
      }
    },
    handleDelete(row) {
      this.$confirm('确定删除此审查记录？', '提示', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }).then(async () => {
        await this.$store.dispatch('contract/deleteHistory', row.id || row.report_id)
        this.$message.success('删除成功')
        this.fetchList()
      }).catch(() => {})
    },
    levelType(level) {
      return { high: 'danger', medium: 'warning', low: 'primary', pass: 'success' }[level] || 'info'
    },
    conclusionType(conclusion) {
      if (!conclusion) return 'info'
      if (conclusion.includes('不通过')) return 'danger'
      if (conclusion.includes('有条件')) return 'warning'
      return 'success'
    }
  }
}
</script>

<style scoped>
.contract-history { height: 100%; display: flex; flex-direction: column; overflow: auto; }
.page-header { background: #fff; padding: 16px 32px; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; }
.page-header h2 { font-size: 20px; margin: 0; color: #333; }
.filter-card { margin: 12px 32px; border-radius: 4px; }
.filter-bar { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.list-card { margin: 0 32px 12px; flex: 1; overflow: auto; }
.pagination-wrap { display: flex; justify-content: flex-end; padding: 12px 0 0; }
.dialog-report { padding: 0 16px; }
.dialog-overview { margin-bottom: 20px; }
.dialog-overview h3 { font-size: 18px; margin: 0 0 12px; }
.dialog-meta { display: flex; gap: 20px; font-size: 13px; color: #666; margin-bottom: 12px; }
.risk-stat { display: flex; gap: 24px; margin-bottom: 12px; }
.risk-item { display: flex; align-items: center; gap: 6px; font-size: 14px; }
.risk-num { font-size: 24px; font-weight: 700; }
.risk-item.high .risk-num { color: #f56c6c; }
.risk-item.medium .risk-num { color: #e6a23c; }
.risk-item.low .risk-num { color: #409eff; }
.dialog-actions { text-align: center; padding: 16px 0; }
.progress-bar { position: relative; height: 20px; background: #ebeef5; border-radius: 10px; overflow: hidden; }
.progress-fill { height: 100%; background: #409eff; border-radius: 10px; transition: width 0.3s; }
.progress-text { position: absolute; top: 0; left: 0; right: 0; line-height: 20px; text-align: center; font-size: 12px; color: #333; }
</style>
