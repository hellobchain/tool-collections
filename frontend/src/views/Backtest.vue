<template>
  <div class="backtest">
    <div class="page-header">
      <h1>策略回测</h1>
      <el-button type="primary" @click="handleRunBacktest" :loading="running" icon="el-icon-video-play">运行回测</el-button>
    </div>

    <el-row :gutter="16" v-if="performance">
      <el-col :span="6">
        <el-card shadow="never" class="perf-card">
          <div class="perf-label">总交易次数</div>
          <div class="perf-value">{{ performance.total_trades }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="perf-card">
          <div class="perf-label">胜率</div>
          <div class="perf-value win">{{ performance.win_rate?.toFixed(1) }}%</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="perf-card">
          <div class="perf-label">总收益率</div>
          <div class="perf-value" :class="performance.total_return_pct >= 0 ? 'win' : 'loss'">{{ performance.total_return_pct?.toFixed(2) }}%</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="perf-card">
          <div class="perf-label">平均收益率</div>
          <div class="perf-value" :class="performance.avg_return_pct >= 0 ? 'win' : 'loss'">{{ performance.avg_return_pct?.toFixed(2) }}%</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="mt-20">
      <div slot="header">回测记录</div>
      <el-input v-model="filterCode" placeholder="股票代码" size="small" class="filter-input" @keyup.enter="fetchResults" />
      <el-button size="small" type="primary" class="ml-10" @click="fetchResults">搜索</el-button>
      <el-table :data="results" stripe v-loading="loading" class="mt-10">
        <el-table-column prop="stock_code" label="代码" width="110" />
        <el-table-column prop="stock_name" label="名称" width="120" />
        <el-table-column prop="action" label="信号" width="70">
          <template slot-scope="s">
            <el-tag :type="s.row.action === 'buy' ? 'success' : s.row.action === 'sell' ? 'danger' : 'warning'" size="mini">{{ s.row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="entry_price" label="入场价" width="90" align="right">
          <template slot-scope="s">{{ s.row.entry_price?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column prop="exit_price" label="出场价" width="90" align="right">
          <template slot-scope="s">{{ s.row.exit_price?.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column prop="return_pct" label="收益率" width="90" align="right">
          <template slot-scope="s">
            <span :class="s.row.return_pct >= 0 ? 'price-up' : 'price-down'">{{ s.row.return_pct?.toFixed(2) }}%</span>
          </template>
        </el-table-column>
        <el-table-column prop="hold_days" label="持有天数" width="80" align="center" />
        <el-table-column prop="analysis_date" label="分析日期" width="100" />
        <el-table-column prop="created_at" label="回测时间" width="160" />
      </el-table>
      <el-pagination
        v-if="total > 0"
        background
        layout="prev, pager, next"
        :total="total"
        :page-size="20"
        class="mt-10 pull-right"
        @current-change="handlePage"
      />
    </el-card>
  </div>
</template>

<script>
import * as stockApi from '@/api/stock'

export default {
  name: 'Backtest',
  data() {
    return {
      running: false,
      loading: false,
      performance: null,
      results: [],
      total: 0,
      page: 1,
      filterCode: ''
    }
  },
  mounted() {
    this.fetchPerformance()
    this.fetchResults()
  },
  methods: {
    async handleRunBacktest() {
      this.running = true
      try {
        await stockApi.runBacktest({})
        this.$message.success('回测完成')
        this.fetchPerformance()
        this.fetchResults()
      } catch { this.$message.error('回测失败') }
      finally { this.running = false }
    },
    async fetchPerformance() {
      try {
        const res = await stockApi.getBacktestPerformance()
        this.performance = res.data.data
      } catch {}
    },
    async fetchResults() {
      this.loading = true
      try {
        const res = await stockApi.getBacktestResults({ page: this.page, limit: 20, code: this.filterCode })
        const d = res.data.data || {}
        this.results = d.list || []
        this.total = d.total || 0
      } finally { this.loading = false }
    },
    handlePage(p) {
      this.page = p
      this.fetchResults()
    }
  }
}
</script>

<style scoped>
.backtest { padding: 20px; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-header h1 { font-size: 22px; margin: 0; }
.perf-card { text-align: center; padding: 8px; }
.perf-label { font-size: 13px; color: #909399; margin-bottom: 8px; }
.perf-value { font-size: 28px; font-weight: bold; color: #303133; }
.perf-value.win { color: #f56c6c; }
.perf-value.loss { color: #67c23a; }
.mt-20 { margin-top: 20px; }
.mt-10 { margin-top: 10px; }
.ml-10 { margin-left: 10px; }
.filter-input { width: 160px; }
.pull-right { float: right; }
.price-up { color: #f56c6c; font-weight: 500; }
.price-down { color: #67c23a; font-weight: 500; }
</style>
