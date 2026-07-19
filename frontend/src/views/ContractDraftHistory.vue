<template>
  <div class="draft-history">
    <div class="page-header">
      <h2>📂 我的起草</h2>
    </div>

    <el-card class="list-card">
      <el-table :data="list" v-loading="loading" stripe style="width:100%">
        <el-table-column prop="file_name" label="模板名称" min-width="180">
          <template slot-scope="{ row }">
            <el-button type="text" @click="viewDetail(row)">{{ row.file_name }}</el-button>
          </template>
        </el-table-column>
        <el-table-column prop="requirements" label="需求摘要" min-width="250" show-overflow-tooltip />
        <el-table-column prop="generated_at" label="生成时间" width="160" />
        <el-table-column prop="content_len" label="草案字数" width="100">
          <template slot-scope="{ row }">{{ row.content_len }}字</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template slot-scope="{ row }">
            <el-tag v-if="row.status==='completed'" type="success" size="mini">已完成</el-tag>
            <el-tag v-else-if="row.status==='failed'" type="danger" size="mini">已失败</el-tag>
            <el-tag v-else-if="row.status==='generating'" type="warning" size="mini">生成中</el-tag>
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
        <el-table-column label="操作" width="120" fixed="right">
          <template slot-scope="{ row }">
            <el-button type="text" size="mini" icon="el-icon-view" @click="viewDetail(row)">查看</el-button>
            <el-button type="text" size="mini" icon="el-icon-delete" style="color:#999" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrap" v-if="total > 0">
        <el-pagination
          @current-change="handlePageChange"
          :current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          small
        />
      </div>
    </el-card>

    <!-- Detail dialog -->
    <el-dialog :visible.sync="detailVisible" fullscreen title="起草详情">
      <div class="dialog-body" v-if="detail">
        <div class="dialog-meta">
          <span>模板：{{ detail.file_name }}</span>
          <span>生成时间：{{ detail.generated_at }}</span>
          <span>状态：
            <el-tag v-if="detail.status==='completed'" type="success" size="mini">已完成</el-tag>
            <el-tag v-else-if="detail.status==='failed'" type="danger" size="mini">已失败</el-tag>
            <el-tag v-else-if="detail.status==='generating'" type="warning" size="mini">生成中</el-tag>
            <span v-else>{{ detail.status }}</span>
          </span>
          <span>进度：{{ detail.progress || 0 }}%</span>
        </div>
        <div class="dialog-section">
          <h4>📋 用户需求</h4>
          <pre class="dialog-pre">{{ detail.requirements }}</pre>
        </div>
        <div class="dialog-section">
          <h4>📄 合同草案</h4>
          <div class="dialog-md" v-html="renderMd(detail.content)"></div>
        </div>
        <div class="dialog-section" v-if="detail.change_log">
          <h4>📝 条款变更说明</h4>
          <div class="dialog-md" v-html="renderMd(detail.change_log)"></div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { marked } from 'marked'
import api from '@/api/index'
import * as contractApi from '@/api/contract'

export default {
  name: 'ContractDraftHistory',
  data() {
    return {
      list: [],
      total: 0,
      page: 1,
      pageSize: 15,
      loading: false,
      detailVisible: false,
      detail: null
    }
  },
  created() {
    this.fetchList()
  },
  methods: {
    renderMd(text) {
      if (!text) return ''
      return marked.parse(text, { breaks: true })
    },
    async fetchList() {
      this.loading = true
      try {
        const res = await contractApi.getDraftHistory({ page: this.page, page_size: this.pageSize } )
        const data = res.data.data || res.data
        this.list = data.list || []
        this.total = data.total || 0
      } catch {} finally {
        this.loading = false
      }
    },
    handlePageChange(val) {
      this.page = val
      this.fetchList()
    },
    async viewDetail(row) {
      try {
        const res = await contractApi.getDraftDetail(row.id)
        this.detail = res.data.data || res.data
        this.detailVisible = true
      } catch {
      }
    },
    handleDelete(row) {
      this.$confirm('确定删除此记录？', '提示', { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }).then(async () => {
        try {
          await contractApi.deleteDraft(row.id)
          this.$message.success('删除成功')
          this.fetchList()
        } catch {
        }
      }).catch(() => {})
    }
  }
}
</script>

<style scoped>
.draft-history { height: 100%; display: flex; flex-direction: column; overflow: auto; }
.page-header { background: #fff; padding: 8px 20px; border-bottom: 1px solid #e4e7ed; flex-shrink: 0; }
.page-header h2 { font-size: 16px; margin: 0; color: #333; }
.list-card { margin: 0; border-top: none; border-radius: 0; flex: 1; overflow: auto; }
.pagination-wrap { display: flex; justify-content: flex-end; padding: 8px 16px; }
.dialog-body { padding: 0 16px; }
.dialog-meta { display: flex; gap: 20px; font-size: 13px; color: #666; margin-bottom: 16px; }
.dialog-section { margin-bottom: 20px; }
.dialog-section h4 { font-size: 15px; margin: 0 0 8px; color: #333; }
.dialog-pre { white-space: pre-wrap; word-break: break-all; line-height: 1.7; font-size: 13px; background: #f5f7fa; padding: 12px; border-radius: 4px; margin: 0; max-height: 50vh; overflow: auto; }
.dialog-md { line-height: 1.8; font-size: 14px; background: #f5f7fa; padding: 12px 16px; border-radius: 4px; max-height: 50vh; overflow: auto; }
.dialog-md >>> h1, .dialog-md >>> h2, .dialog-md >>> h3, .dialog-md >>> h4 { margin: 12px 0 6px; }
.dialog-md >>> p { margin: 6px 0; }
.dialog-md >>> ul, .dialog-md >>> ol { padding-left: 20px; }
.dialog-md >>> code { background: #e8e8e8; padding: 1px 4px; border-radius: 2px; font-size: 13px; }
.dialog-md >>> pre { background: #e8e8e8; padding: 10px; border-radius: 4px; overflow: auto; }
.progress-bar { position: relative; height: 18px; background: #ebeef5; border-radius: 9px; overflow: hidden; }
.progress-fill { height: 100%; background: #409eff; border-radius: 9px; transition: width 0.3s; }
.progress-text { position: absolute; top: 0; left: 0; right: 0; line-height: 18px; text-align: center; font-size: 11px; color: #333; }
</style>
