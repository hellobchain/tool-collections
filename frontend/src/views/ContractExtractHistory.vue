<template>
  <div class="extract-history">
    <div class="page-header">
      <h2>📂 我的提取</h2>
    </div>

    <el-card class="filter-card">
      <div class="filter-bar">
        <el-input v-model="searchText" placeholder="搜索任务名称..." size="small" style="width:220px" prefix-icon="el-icon-search" clearable @clear="handleSearch" />
        <el-date-picker v-model="dateRange" type="daterange" range-separator="~" start-placeholder="创建开始" end-placeholder="创建结束" size="small" value-format="yyyy-MM-dd" style="width:260px" @change="handleSearch" />
        <el-button type="primary" size="small" icon="el-icon-search" @click="handleSearch">搜索</el-button>
      </div>
    </el-card>

    <el-card class="list-card">
      <el-table :data="list" v-loading="loading" stripe style="width:100%">
        <el-table-column prop="task_name" label="任务名称" min-width="200">
          <template slot-scope="{ row }">
            <EllipsisCell :content="row.task_name" maxWidth="200px" />
          </template>
        </el-table-column>
        <el-table-column prop="file_count" label="文件数" width="80" />
        <el-table-column prop="field_count" label="提取字段数" width="110" />
        <el-table-column label="状态" width="90">
          <template slot-scope="{ row }">
            <el-tag v-if="row.status==='completed'" type="success" size="mini">已完成</el-tag>
            <el-tag v-else-if="row.status==='failed'" type="danger" size="mini">失败</el-tag>
            <el-tag v-else-if="row.status==='extracting'" type="warning" size="mini">提取中</el-tag>
            <span v-else>{{ row.status }}</span>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="140">
          <template slot-scope="{ row }">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: (row.progress || 0) + '%' }"></div>
              <span class="progress-text">{{ row.progress || 0 }}%</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="150" />
        <el-table-column label="操作" width="170" fixed="right">
          <template slot-scope="{ row }">
            <el-button type="text" size="mini" icon="el-icon-view" @click="viewDetail(row)">查看</el-button>
            <el-button type="text" size="mini" icon="el-icon-download" @click="handleExport(row)">导出</el-button>
            <el-button type="text" size="mini" icon="el-icon-delete" style="color:#999" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap" v-if="total > 0">
        <el-pagination @current-change="handlePageChange" :current-page="page" :page-size="pageSize" :total="total" layout="total, prev, pager, next" small />
      </div>
    </el-card>

    <el-dialog :visible.sync="detailVisible" fullscreen title="提取详情">
      <div class="dialog-body" v-if="detail">
        <div class="dialog-meta">
          <span>任务：{{ detail.task_name }}</span>
          <span>状态：
            <el-tag v-if="detail.status==='completed'" type="success" size="mini">已完成</el-tag>
            <el-tag v-else-if="detail.status==='failed'" type="danger" size="mini">失败</el-tag>
            <el-tag v-else-if="detail.status==='extracting'" type="warning" size="mini">提取中</el-tag>
            <span v-else>{{ detail.status }}</span>
          </span>
        </div>
        <el-table :data="detail.results || []" stripe border size="small" style="width:100%">
          <el-table-column prop="file_name" label="文件名" min-width="160" fixed />
          <el-table-column v-for="f in (detail.fields || [])" :key="f.name" :label="f.name" min-width="140">
            <template slot-scope="{ row }">
              <EllipsisCell :content="row.data ? row.data[f.name] : '-'" maxWidth="140px" />
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { getExtractHistory, getExtractResult, deleteExtractTask, exportExtractResult } from '@/api/contract'

export default {
  name: 'ContractExtractHistory',
  components: {
    EllipsisCell: () => import('@/components/EllipsisCell')
  },
  data() {
    return {
      list: [], total: 0, page: 1, pageSize: 15, loading: false,
      detailVisible: false, detail: null,
      searchText: '', dateRange: null
    }
  },
  created() { this.fetchList() },
  methods: {
    async fetchList() {
      this.loading = true
      try {
        const params = { page: this.page, page_size: this.pageSize }
        if (this.searchText) params.keyword = this.searchText
        if (this.dateRange && this.dateRange.length === 2) {
          params.date_from = this.dateRange[0]
          params.date_to = this.dateRange[1]
        }
        const res = await getExtractHistory(params)
        const d = res.data.data || res.data
        this.list = d.list || []; this.total = d.total || 0
      } catch {} finally { this.loading = false }
    },
    handleSearch() { this.page = 1; this.fetchList() },
    handlePageChange(val) { this.page = val; this.fetchList() },
    async viewDetail(row) {
      try {
        const res = await getExtractResult(row.id)
        this.detail = res.data.data || res.data
        this.detailVisible = true
      } catch {  }
    },
    async handleExport(row) {
      try {
        const res = await exportExtractResult(row.id)
        const url = URL.createObjectURL(res.data)
        const a = document.createElement('a'); a.href = url; a.download = `提取结果_${row.task_name}.xlsx`
        document.body.appendChild(a); a.click(); document.body.removeChild(a); URL.revokeObjectURL(url)
      } catch { }
    },
    handleDelete(row) {
      this.$confirm('确定删除？', '提示', { type: 'warning' }).then(async () => {
        try { await deleteExtractTask(row.id); this.$message.success('删除成功'); this.fetchList() }
        catch {  }
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
.extract-history { height: 100%; display: flex; flex-direction: column; overflow: auto; }
.page-header { background: #fff; padding: 8px 20px; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; }
.page-header h2 { font-size: 16px; margin: 0; color: #333; }
.filter-card { margin: 0; border-top: none; border-radius: 0; }
.filter-bar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; padding: 8px 16px; }
.list-card { margin: 0; border-top: none; border-radius: 0; flex: 1; overflow: auto; }
.pagination-wrap { display: flex; justify-content: flex-end; padding: 8px 16px; }
.dialog-body { padding: 0 16px; }
.dialog-meta { display: flex; gap: 20px; font-size: 13px; color: #666; margin-bottom: 16px; }
.progress-bar { position: relative; height: 18px; background: #ebeef5; border-radius: 9px; overflow: hidden; }
.progress-fill { height: 100%; background: #409eff; border-radius: 9px; transition: width 0.3s; }
.progress-text { position: absolute; top: 0; left: 0; right: 0; line-height: 18px; text-align: center; font-size: 11px; color: #333; }
</style>
