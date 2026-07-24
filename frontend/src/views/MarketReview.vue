<template>
  <div>
    <div class="page-header">
      <h2>📊 大盘复盘</h2>
      <p class="page-desc">AI自动分析今日A股市场整体表现</p>
      <el-button type="primary" @click="handleTrigger" :loading="loading" icon="el-icon-refresh" size="small" style="margin-top:8px;">触发复盘</el-button>
    </div>

    <el-card shadow="never" class="compact-card" v-if="review">
      <div class="review-date">复盘日期: {{ review.date }}</div>
      <el-divider style="margin:8px 0;" />
      <div class="section"><h4>主要指数</h4>
        <el-row :gutter="8">
          <el-col :span="8" v-for="(idx, name) in review.indices" :key="name">
            <div class="index-card"><div class="index-name">{{ idx.name }}</div><div class="index-value">{{ idx.value }}</div><div class="index-change" :class="idx.change_pct >= 0 ? 'up' : 'down'">{{ idx.change_pct >= 0 ? '+' : '' }}{{ idx.change_pct }}%</div></div>
          </el-col>
        </el-row>
      </div>
      <div class="section"><h4>市场概况</h4>
        <el-row :gutter="8">
          <el-col :span="6"><div class="stat-item"><label>上涨</label><span class="up">{{ review.market_overview?.advance }}</span></div></el-col>
          <el-col :span="6"><div class="stat-item"><label>下跌</label><span class="down">{{ review.market_overview?.decline }}</span></div></el-col>
          <el-col :span="6"><div class="stat-item"><label>涨停</label><span class="up">{{ review.market_overview?.limit_up }}</span></div></el-col>
          <el-col :span="6"><div class="stat-item"><label>跌停</label><span class="down">{{ review.market_overview?.limit_down }}</span></div></el-col>
        </el-row>
      </div>
      <div class="section"><h4>热点板块</h4>
        <el-tag v-for="s in review.hot_sectors" :key="s" class="sector-tag" type="warning" size="small">{{ s }}</el-tag>
      </div>
      <div class="section"><h4>分析点评</h4><p class="analysis-text">{{ review.analysis }}</p></div>
    </el-card>
    <el-empty v-else-if="!loading" description="点击"触发复盘"按钮生成大盘分析" />
  </div>
</template>

<script>
export default {
  name: 'MarketReview',
  data() { return { loading: false, review: null } },
  methods: {
    async handleTrigger() {
      this.loading = true
      try { const data = await this.$store.dispatch('stock/triggerMarketReview', { send_notification: false }); this.review = data?.report || data; this.$message.success('大盘复盘完成') }
      catch { this.$message.error('复盘失败') }
      finally { this.loading = false }
    }
  }
}
</script>

<style scoped>
.page-header { padding: 12px 0 6px; }
.page-header h2 { font-size: 18px; margin: 0; }
.page-desc { font-size: 12px; color: #909399; margin: 2px 0 0; }
.compact-card >>> .el-card__body { padding: 10px 14px; }
.section { margin: 12px 0; }
.section h4 { font-size: 14px; margin: 0 0 8px; padding-left: 8px; border-left: 2px solid #409eff; }
.review-date { font-size: 14px; font-weight: bold; }
.index-card { text-align: center; padding: 10px; background: #f5f7fa; border-radius: 6px; }
.index-name { font-size: 12px; color: #909399; }
.index-value { font-size: 22px; font-weight: bold; }
.index-change { font-size: 14px; }
.up { color: #f56c6c; }
.down { color: #67c23a; }
.stat-item { text-align: center; padding: 10px; background: #f5f7fa; border-radius: 6px; }
.stat-item label { display: block; font-size: 11px; color: #909399; }
.stat-item span { font-size: 20px; font-weight: bold; }
.sector-tag { margin: 2px 3px; }
.analysis-text { font-size: 13px; line-height: 1.7; color: #606266; }
</style>
