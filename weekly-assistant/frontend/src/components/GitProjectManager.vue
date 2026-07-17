<template>
  <el-dialog title="🔗 Git 项目管理" :visible="visible" width="600px" :before-close="handleClose" @open="handleOpen">
    <div class="hint" style="margin-bottom:12px;font-size:13px;color:#909399;">
      管理GitLab/Bitbucket等代码仓库连接，拉取commit自动生成碎片。
    </div>

    <div class="project-list" v-if="projects.length > 0">
      <div v-for="p in projects" :key="p.id" class="project-item">
        <div class="project-info">
          <div class="project-name">{{ p.project_name }}</div>
          <div class="project-url">{{ p.base_url }}/{{ p.project_id }}</div>
        </div>
        <div class="project-actions">
          <el-button type="text" icon="el-icon-edit" size="mini" @click="editProject(p)" />
          <el-button type="text" icon="el-icon-delete" size="mini" style="color:#f56c6c" @click="removeProject(p)" />
        </div>
      </div>
    </div>
    <div v-else style="text-align:center;padding:24px;color:#ccc;">
      <p>还没有关联的项目</p>
    </div>

    <div class="project-pagination" v-if="total > 0">
      <span class="project-pagination-info">共 {{ total }} 条</span>
      <el-pagination
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        background
        small
        @current-change="onPageChange"
      />
    </div>

    <div class="create-area">
      <el-button type="primary" size="small" @click="showForm = true; editingId = ''; form = { project_id: '', project_name: '', base_url: '', token: '', branch: 'master' }">
        <i class="el-icon-plus" /> 添加项目
      </el-button>
    </div>

    <el-dialog :title="editingId ? '编辑项目' : '添加项目'" :visible.sync="showForm" width="500px" append-to-body>
      <el-form label-width="100px" size="small">
        <el-form-item label="项目名称" required>
          <el-input v-model="form.project_name" placeholder="如：用户中心后端" />
        </el-form-item>
        <el-form-item label="项目ID" required>
          <el-input v-model="form.project_id" placeholder="GitLab 项目ID（数字）" />
        </el-form-item>
        <el-form-item label="GitLab地址" required>
          <el-input v-model="form.base_url" placeholder="https://gitlab.company.com" />
        </el-form-item>
        <el-form-item label="Token" required>
          <el-input v-model="form.token" type="password" placeholder="Personal Access Token" show-password />
        </el-form-item>
        <el-form-item label="分支">
          <el-input v-model="form.branch" placeholder="master" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="showForm = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveProject">保存</el-button>
      </span>
    </el-dialog>
  </el-dialog>
</template>

<script>
import api from '@/api'

export default {
  props: {
    visible: { type: Boolean, default: false },
    onSelect: { type: Function, default: null }
  },
  data() {
    return {
      projects: [],
      total: 0,
      page: 1,
      pageSize: 10,
      showForm: false,
      editingId: '',
      saving: false,
      form: { project_id: '', project_name: '', base_url: '', token: '', branch: 'master' }
    }
  },
  methods: {
    handleOpen() {
      this.page = 1
      this.loadProjects()
    },
    async loadProjects() {
      const res = await api.get('/api/git-projects', { params: { page: this.page, page_size: this.pageSize } })
      if (res.data.code === 0) {
        this.projects = res.data.data.list
        this.total = res.data.data.total
      }
    },
    onPageChange(page) {
      this.page = page
      this.loadProjects()
    },
    selectProject(p) {
      this.$emit('select', p)
    },
    editProject(p) {
      this.editingId = p.id
      this.form = {
        project_id: p.project_id,
        project_name: p.project_name,
        base_url: p.base_url,
        token: p.token,
        branch: p.branch || 'master'
      }
      this.showForm = true
    },
    async saveProject() {
      if (!this.form.project_id || !this.form.project_name || !this.form.base_url || !this.form.token) {
        this.$message.warning('请填写完整信息')
        return
      }
      this.saving = true
      try {
        const url = this.editingId ? `/api/git-projects/${this.editingId}` : '/api/git-projects'
        const method = this.editingId ? 'put' : 'post'
        const res = await api[method](url, this.form)
        if (res.data.code === 0) {
          this.$message.success(this.editingId ? '更新成功' : '添加成功')
          this.showForm = false
          this.loadProjects()
        }
      } finally {
        this.saving = false
      }
    },
    async removeProject(p) {
      try {
        await this.$confirm(`确定删除项目"${p.project_name}"吗？`, '确认删除', { type: 'warning' })
      } catch { return }
      const res = await api.delete(`/api/git-projects/${p.id}`)
      if (res.data.code === 0) {
        this.$message.success('已删除')
        this.loadProjects()
      }
    },
    handleClose() {
      this.$emit('update:visible', false)
    }
  }
}
</script>

<style scoped>
.project-list {
  max-height: 360px;
  overflow-y: auto;
}
.project-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.2s;
}
.project-item:hover {
  background: #f8f9fa;
}
.project-info {
  flex: 1;
  min-width: 0;
}
.project-name {
  font-weight: 500;
  font-size: 14px;
}
.project-url {
  font-size: 12px;
  color: #999;
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.project-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.create-area {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #eee;
  text-align: center;
}
.project-pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  padding: 12px 0 0;
}
.project-pagination-info {
  font-size: 13px;
  color: #909399;
}
</style>
