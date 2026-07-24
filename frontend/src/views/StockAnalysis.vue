<template>
  <div>
    <div class="page-header">
      <h2>📈 股票智能分析</h2>
      <p class="page-desc">输入股票代码，获取AI智能分析报告与实时行情</p>
    </div>

    <div class="page-header-actions">
      <el-input v-model="stockCodeInput" placeholder="输入股票代码/名称（如600519、AAPL）" class="stock-input" @keyup.enter="handleAnalyze" :disabled="analysisLoading" />
      <el-select v-model="reportType" class="report-type-select" size="small">
        <el-option label="精简" value="simple" /><el-option label="完整" value="detailed" /><el-option label="简洁" value="brief" />
      </el-select>
      <el-button type="primary" @click="handleAnalyze" :loading="analysisLoading" icon="el-icon-search" size="small">分析</el-button>
    </div>

    <el-tabs v-model="activeTab" class="stock-tabs">
      <el-tab-pane label="行情与K线" name="quote">
        <el-row :gutter="10">
          <el-col :span="7">
            <el-card shadow="never" class="compact-card">
              <div slot="header" class="compact-header"><span>实时行情</span></div>
              <div class="compact-body">
                <el-input v-model="quoteCode" placeholder="输入股票代码" size="mini" class="mb-5" @keyup.enter="handleFetchQuote" />
                <el-button type="primary" size="mini" @click="handleFetchQuote" :loading="quoteLoading">查询</el-button>
                <div v-if="quote" class="quote-info mt-8">
                  <div class="quote-name">{{ quote.stock_name }}<span class="quote-code">({{ quote.stock_code }})</span></div>
                  <div class="quote-price" :class="priceClass(quote.change_percent)">{{ quote.current_price }}</div>
                  <div class="quote-change" :class="priceClass(quote.change_percent)">{{ quote.change_percent >= 0 ? '+' : '' }}{{ quote.change_percent }}%</div>
                  <div class="quote-grid">
                    <div><label>开盘</label><span>{{ quote.open }}</span></div>
                    <div><label>最高</label><span>{{ quote.high }}</span></div>
                    <div><label>最低</label><span>{{ quote.low }}</span></div>
                    <div><label>昨收</label><span>{{ quote.prev_close }}</span></div>
                    <div><label>成交量</label><span>{{ formatVolume(quote.volume) }}</span></div>
                    <div><label>成交额</label><span>{{ formatVolume(quote.amount) }}</span></div>
                  </div>
                </div>
              </div>
            </el-card>
          </el-col>
          <el-col :span="17">
            <el-card shadow="never" class="compact-card">
              <div slot="header" class="compact-header">
                <span>K线图</span>
                <span class="pull-right">
                  <el-radio-group v-model="klinePeriod" size="mini" @change="handleFetchKline">
                    <el-radio-button label="daily">日线</el-radio-button>
                    <el-radio-button label="weekly">周线</el-radio-button>
                    <el-radio-button label="monthly">月线</el-radio-button>
                  </el-radio-group>
                </span>
              </div>
              <div>
                <el-table :data="klineData" stripe size="mini" height="380" v-loading="klineLoading">
                  <el-table-column prop="date" label="日期" width="95"/>
                  <el-table-column prop="open" label="开盘" width="85" align="right"><template slot-scope="s">{{ s.row.open?.toFixed(2) }}</template></el-table-column>
                  <el-table-column prop="high" label="最高" width="85" align="right"><template slot-scope="s">{{ s.row.high?.toFixed(2) }}</template></el-table-column>
                  <el-table-column prop="low" label="最低" width="85" align="right"><template slot-scope="s">{{ s.row.low?.toFixed(2) }}</template></el-table-column>
                  <el-table-column prop="close" label="收盘" width="85" align="right"><template slot-scope="s">{{ s.row.close?.toFixed(2) }}</template></el-table-column>
                  <el-table-column prop="change_percent" label="涨跌幅" width="85" align="right">
                    <template slot-scope="s"><span :class="priceClass(s.row.change_percent)">{{ s.row.change_percent?.toFixed(2) }}%</span></template>
                  </el-table-column>
                  <el-table-column prop="volume" label="成交量" min-width="90" align="right"><template slot-scope="s">{{ formatVolume(s.row.volume) }}</template></el-table-column>
                </el-table>
              </div>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane label="分析报告" name="report">
        <el-row :gutter="10">
          <el-col :span="6">
            <el-card shadow="never" class="compact-card">
              <div slot="header" class="compact-header">自选股 <el-button size="mini" type="text" @click="showAddWatchlist = true"><i class="el-icon-plus"></i>添加</el-button></div>
              <div v-loading="watchlistLoading">
                <div v-for="code in watchlist" :key="code" class="watchlist-item"><span class="watchlist-code" @click="analyzeStock(code)">{{ code }}</span><el-button type="text" size="mini" class="watchlist-btn" @click="handleRemoveWatchlist(code)">×</el-button></div>
                <el-empty v-if="!watchlist.length && !watchlistLoading" description="暂无自选" />
              </div>
            </el-card>
            <el-card shadow="never" class="compact-card mt-5">
              <div slot="header" class="compact-header">导入股票</div>
              <el-upload action="#" :auto-upload="false" :show-file-list="false" accept=".csv,.xlsx,.xls,.txt" :on-change="handleImportFile"><el-button size="mini" type="primary">上传文件</el-button></el-upload>
              <el-input v-model="importText" type="textarea" :rows="2" placeholder="粘贴股票代码，每行一个" class="mt-5" />
              <el-button size="mini" class="mt-5" @click="handleImportText">解析导入</el-button>
            </el-card>
          </el-col>
          <el-col :span="18">
            <el-card shadow="never" class="compact-card">
              <div slot="header" class="compact-header">分析结果</div>
              <div v-if="report" class="report-content">
                <div class="report-head">
                  <h3>{{ report.meta?.stock_name }} <small>{{ report.meta?.stock_code }}</small></h3>
                  <div class="report-tags">
                    <el-tag :type="reportTagType(report.summary?.action)" size="small">{{ report.summary?.action_label }}</el-tag>
                    <span class="report-score">评分 {{ report.summary?.sentiment_score }}/100</span>
                  </div>
                </div>
                <div class="report-section"><h4>操作建议</h4><p>{{ report.summary?.operation_advice }}</p></div>
                <div class="report-section"><h4>趋势预测</h4><p>{{ report.summary?.trend_prediction }}</p></div>
                <div class="report-section"><h4>分析摘要</h4><p>{{ report.summary?.analysis_summary }}</p></div>
                <div v-if="report.strategy" class="report-section"><h4>策略价格</h4>
                  <el-row :gutter="8">
                    <el-col :span="6"><div class="strategy-item">理想买入<br/><strong>{{ report.strategy.ideal_buy }}</strong></div></el-col>
                    <el-col :span="6"><div class="strategy-item">次级买入<br/><strong>{{ report.strategy.secondary_buy }}</strong></div></el-col>
                    <el-col :span="6"><div class="strategy-item stop">止损<br/><strong>{{ report.strategy.stop_loss }}</strong></div></el-col>
                    <el-col :span="6"><div class="strategy-item profit">止盈<br/><strong>{{ report.strategy.take_profit }}</strong></div></el-col>
                  </el-row>
                </div>
              </div>
              <el-empty v-else-if="!analysisLoading" description="点击"分析"按钮开始分析" />
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>

      <el-tab-pane label="个股栏" name="stockbar">
        <el-card shadow="never" class="compact-card">
          <el-table :data="stockBar" stripe v-loading="stockBarLoading" @row-click="handleStockBarClick" size="mini">
            <el-table-column prop="stock_code" label="代码" width="100" />
            <el-table-column prop="stock_name" label="名称" width="120" />
            <el-table-column prop="action_label" label="建议" width="80"><template slot-scope="s"><el-tag :type="s.row.action === 'buy' ? 'success' : s.row.action === 'sell' ? 'danger' : 'warning'" size="mini">{{ s.row.action_label }}</el-tag></template></el-table-column>
            <el-table-column prop="sentiment_score" label="评分" width="60" align="center" />
            <el-table-column prop="model_used" label="模型" min-width="110" />
            <el-table-column prop="analysis_count" label="分析次数" width="75" align="center" />
            <el-table-column prop="last_analysis_time" label="最近分析" width="150" />
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="历史报告" name="history">
        <el-card shadow="never" class="compact-card">
          <div class="history-filter mb-5">
            <el-input v-model="historyFilter.code" placeholder="股票代码" size="mini" class="filter-item" />
            <el-date-picker v-model="historyFilter.startDate" type="date" placeholder="开始日期" size="mini" class="filter-item" value-format="yyyy-MM-dd" />
            <el-date-picker v-model="historyFilter.endDate" type="date" placeholder="结束日期" size="mini" class="filter-item" value-format="yyyy-MM-dd" />
            <el-button size="mini" type="primary" @click="handleHistorySearch">搜索</el-button>
          </div>
          <el-table :data="historyList" stripe v-loading="historyLoading" @row-click="handleHistoryClick" size="mini">
            <el-table-column prop="stock_code" label="代码" width="100" />
            <el-table-column prop="stock_name" label="名称" width="100" />
            <el-table-column prop="action_label" label="建议" width="65"><template slot-scope="s"><el-tag :type="s.row.action === 'buy' ? 'success' : s.row.action === 'sell' ? 'danger' : 'warning'" size="mini">{{ s.row.action_label }}</el-tag></template></el-table-column>
            <el-table-column prop="sentiment_score" label="评分" width="55" align="center" />
            <el-table-column prop="operation_advice" label="操作建议" min-width="140" show-overflow-tooltip />
            <el-table-column prop="analysis_summary" label="摘要" min-width="180" show-overflow-tooltip />
            <el-table-column prop="created_at" label="时间" width="150" />
          </el-table>
          <el-pagination v-if="historyTotal > 0" background layout="prev, pager, next" :total="historyTotal" :page-size="20" small class="mt-5 pull-right" @current-change="handleHistoryPage" />
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <el-dialog title="添加自选股" :visible.sync="showAddWatchlist" width="350px">
      <el-input v-model="newWatchlistCode" placeholder="输入股票代码" @keyup.enter="handleAddWatchlist" />
      <span slot="footer"><el-button @click="showAddWatchlist = false">取消</el-button><el-button type="primary" @click="handleAddWatchlist">添加</el-button></span>
    </el-dialog>

    <el-dialog title="分析报告详情" :visible.sync="showReportDetail" width="700px" top="5vh">
      <div v-if="currentReport" class="report-detail-content">
        <h3>{{ currentReport.stock_name }} ({{ currentReport.stock_code }})</h3>
        <el-tag :type="currentReport.action === 'buy' ? 'success' : 'danger'" size="small">{{ currentReport.action_label }}</el-tag>
        <div>综合评分: {{ currentReport.sentiment_score }}/100</div>
        <el-divider />
        <h4>操作建议</h4><p>{{ currentReport.operation_advice }}</p>
        <h4>趋势预测</h4><p>{{ currentReport.trend_prediction }}</p>
        <h4>分析摘要</h4><p>{{ currentReport.analysis_summary }}</p>
      </div>
      <span slot="footer"><el-button @click="showReportDetail = false">关闭</el-button></span>
    </el-dialog>
  </div>
</template>

<script>
import { mapState, mapActions } from 'vuex'

export default {
  name: 'StockAnalysis',
  data() {
    return {
      activeTab: 'quote', stockCodeInput: '', reportType: 'detailed', analysisLoading: false, report: null,
      quoteCode: '600519', quote: null, quoteLoading: false, klinePeriod: 'daily', kline: null, klineLoading: false,
      showAddWatchlist: false, newWatchlistCode: '', importText: '', showReportDetail: false, currentReport: null,
      historyFilter: { code: '', startDate: '', endDate: '' }, historyPage: 1
    }
  },
  computed: {
    ...mapState('stock', ['watchlist', 'watchlistLoading', 'stockBar', 'stockBarLoading', 'historyList', 'historyTotal', 'historyLoading']),
    klineData() { return this.kline?.data || [] }
  },
  mounted() {
    this.fetchWatchlist(); this.fetchStockBar({ limit: 200 }); this.fetchHistoryList({ page: 1, limit: 20 }); this.handleFetchQuote()
  },
  methods: {
    ...mapActions('stock', ['fetchWatchlist', 'addWatchlist', 'removeWatchlist', 'fetchStockBar', 'fetchHistoryList']),
    async handleAnalyze() {
      if (!this.stockCodeInput) return; this.analysisLoading = true; this.report = null
      try {
        const res = await this.$store.dispatch('stock/triggerAnalysis', { stock_code: this.stockCodeInput, report_type: this.reportType, async_mode: false })
        this.report = res?.report || null; this.activeTab = 'report'
        if (this.report) { this.$message.success('分析完成'); this.fetchWatchlist(); this.fetchHistoryList({ page: 1, limit: 20 }) }
      } catch { this.$message.error('分析失败') } finally { this.analysisLoading = false }
    },
    analyzeStock(code) { this.stockCodeInput = code; this.handleAnalyze() },
    async handleFetchQuote() {
      if (!this.quoteCode) return; this.quoteLoading = true
      try { const res = await this.$store.dispatch('stock/fetchQuote', this.quoteCode); this.quote = res } finally { this.quoteLoading = false }
    },
    async handleFetchKline() {
      if (!this.quoteCode) return; this.klineLoading = true
      try { const res = await this.$store.dispatch('stock/fetchKline', { code: this.quoteCode, period: this.klinePeriod, days: 60 }); this.kline = res } finally { this.klineLoading = false }
    },
    async handleAddWatchlist() { if (!this.newWatchlistCode) return; await this.addWatchlist(this.newWatchlistCode.trim()); this.newWatchlistCode = ''; this.showAddWatchlist = false; this.$message.success('已添加') },
    async handleRemoveWatchlist(code) { await this.removeWatchlist(code) },
    handleImportFile(file) { const r = new FileReader(); r.onload = e => { this.importText = e.target.result }; r.readAsText(file.raw) },
    async handleImportText() {
      if (!this.importText) return
      try {
        const res = await this.$store.dispatch('stock/importStockCodesText', this.importText)
        const codes = res?.data?.data?.codes || []
        for (const c of codes.slice(0, 50)) await this.addWatchlist(c)
        this.$message.success(`导入 ${codes.length} 只股票`)
      } catch { this.$message.error('导入失败') }
    },
    handleStockBarClick(row) { this.quoteCode = row.stock_code; this.activeTab = 'quote'; this.handleFetchQuote(); this.handleFetchKline() },
    handleHistorySearch() { this.historyPage = 1; this.fetchHistoryList({ page: 1, limit: 20, stock_code: this.historyFilter.code, start_date: this.historyFilter.startDate, end_date: this.historyFilter.endDate }) },
    handleHistoryPage(p) { this.historyPage = p; this.fetchHistoryList({ page: p, limit: 20, stock_code: this.historyFilter.code }) },
    handleHistoryClick(row) { this.currentReport = row; this.showReportDetail = true },
    priceClass(v) { if (v > 0) return 'price-up'; if (v < 0) return 'price-down'; return 'price-flat' },
    reportTagType(a) { if (a === 'buy') return 'success'; if (a === 'sell') return 'danger'; return 'warning' },
    formatVolume(v) { if (!v) return '-'; if (v >= 1e8) return (v / 1e8).toFixed(2) + '亿'; if (v >= 1e4) return (v / 1e4).toFixed(2) + '万'; return '' + v }
  }
}
</script>

<style scoped>
.page-header { padding: 16px 0 4px; }
.page-header h2 { font-size: 18px; margin: 0; color: #303133; }
.page-desc { font-size: 12px; color: #909399; margin: 4px 0 0; }
.page-header-actions { display: flex; gap: 6px; align-items: center; margin: 8px 0; }
.stock-input { width: 240px; }
.report-type-select { width: 90px; }
.stock-tabs { background: #fff; border-radius: 4px; }
.stock-tabs >>> .el-tabs__header { margin-bottom: 8px; }
.stock-tabs >>> .el-tabs__content { padding: 0; }
.stock-tabs >>> .el-tab-pane { padding: 0; }
.compact-card >>> .el-card__header { padding: 8px 12px; }
.compact-card >>> .el-card__body { padding: 10px 12px; }
.compact-header { display: flex; justify-content: space-between; align-items: center; font-size: 13px; font-weight: 500; }
.mb-5 { margin-bottom: 5px; }
.mt-5 { margin-top: 5px; }
.mt-8 { margin-top: 8px; }
.pull-right { float: right; }
.price-up { color: #f56c6c; }
.price-down { color: #67c23a; }
.price-flat { color: #909399; }
.quote-info { font-size: 13px; }
.quote-name { font-weight: bold; font-size: 14px; }
.quote-code { font-size: 11px; color: #909399; font-weight: normal; }
.quote-price { font-size: 30px; font-weight: bold; margin: 4px 0; }
.quote-change { font-size: 15px; margin-bottom: 8px; }
.quote-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 4px; }
.quote-grid div { display: flex; justify-content: space-between; padding: 2px 0; border-bottom: 1px solid #f0f0f0; font-size: 12px; }
.quote-grid label { color: #909399; }
.watchlist-item { display: flex; justify-content: space-between; align-items: center; padding: 4px 0; border-bottom: 1px solid #f5f5f5; cursor: pointer; font-size: 13px; }
.watchlist-item:hover { background: #f9f9f9; }
.watchlist-code { font-family: monospace; }
.watchlist-btn { color: #c0c4cc; }
.watchlist-btn:hover { color: #f56c6c; }
.history-filter { display: flex; gap: 6px; flex-wrap: wrap; }
.filter-item { width: 130px; }
.report-content { font-size: 13px; }
.report-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.report-head h3 { margin: 0; font-size: 16px; }
.report-head h3 small { font-size: 12px; color: #909399; font-weight: normal; }
.report-tags { display: flex; gap: 8px; align-items: center; }
.report-score { font-weight: bold; color: #409eff; font-size: 14px; }
.report-section { margin: 10px 0; }
.report-section h4 { font-size: 13px; margin: 0 0 4px; padding-left: 6px; border-left: 2px solid #409eff; }
.report-section p { font-size: 12px; line-height: 1.6; color: #606266; margin: 0; }
.strategy-item { text-align: center; padding: 8px; border-radius: 6px; background: #f5f7fa; font-size: 12px; }
.strategy-item strong { display: block; margin-top: 2px; font-size: 14px; color: #409eff; }
.strategy-item.stop strong { color: #f56c6c; }
.strategy-item.profit strong { color: #67c23a; }
.report-detail-content { max-height: 60vh; overflow-y: auto; font-size: 13px; }
</style>
