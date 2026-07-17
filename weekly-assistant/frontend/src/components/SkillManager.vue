<template>
  <el-dialog title="🎯 技能/角色管理" :visible="visible" width="700px" :before-close="handleClose" @open="handleOpen">
    <div class="skill-hint" style="margin-bottom: 12px; font-size: 13px; color: #909399;">
      定义你的技术技能或角色定位，AI生成周报时会自动融入这些信息，让周报更贴合你的个人标签。
    </div>

    <!-- 技能列表 -->
    <div class="skill-list">
      <div v-for="s in skills" :key="s.id" class="skill-item">
        <div class="skill-info">
          <div class="skill-header">
            <span class="skill-name">{{ s.name }}</span>
            <el-tag v-if="!s.is_active" size="mini" type="danger">已禁用</el-tag>
          </div>
          <div class="skill-desc">{{ s.description }}</div>
        </div>
        <div class="skill-actions">
          <el-switch v-model="s.is_active" size="mini" @change="toggleSkill(s)" />
          <el-button type="text" size="small" @click="editSkill(s)">编辑</el-button>
          <el-button type="text" size="small" style="color:#f56c6c" @click="removeSkill(s)">删除</el-button>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div class="skill-pagination" v-if="total > 0">
      <span class="skill-pagination-info">共 {{ total }} 条</span>
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

    <!-- 添加 -->
    <div class="create-area">
      <el-button type="primary" size="small" @click="showCreate = true">
        <i class="el-icon-plus" /> 添加技能
      </el-button>
    </div>

    <!-- 编辑弹窗 -->
    <el-dialog title="编辑技能" :visible.sync="showEdit" width="500px" append-to-body>
      <el-form label-width="80px">
        <el-form-item label="技能名称">
          <el-input v-model="editData.name" placeholder="如：后端开发" />
        </el-form-item>
        <el-form-item label="技能描述">
          <el-input v-model="editData.description" type="textarea" :rows="3" placeholder="如：精通Go/Python，擅长高并发微服务架构设计" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="showEdit = false">取消</el-button>
        <el-button type="primary" @click="saveEdit">保存</el-button>
      </span>
    </el-dialog>

    <!-- 创建弹窗 -->
    <el-dialog title="添加技能" :visible.sync="showCreate" width="500px" append-to-body>
      <el-form label-width="80px">
        <el-form-item label="技能名称" required>
          <el-input v-model="createData.name" placeholder="如：后端开发" />
        </el-form-item>
        <el-form-item label="技能描述" required>
          <el-input v-model="createData.description" type="textarea" :rows="3" placeholder="如：精通Go/Python，擅长高并发微服务架构设计" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" @click="createSkill">添加</el-button>
      </span>
    </el-dialog>
  </el-dialog>
</template>

<script>
import api from '@/api'

export default {
  props: {
    visible: { type: Boolean, default: false }
  },
  data() {
    return {
      skills: [],
      total: 0,
      page: 1,
      pageSize: 10,
      showEdit: false,
      showCreate: false,
      editData: { id: '', name: '', description: '' },
      createData: { name: '', description: '' }
    }
  },
  methods: {
    handleOpen() {
      this.page = 1
      this.loadSkills()
    },
    async loadSkills() {
      const res = await api.get(`/weekly-assistant/skills`, { params: { page: this.page, page_size: this.pageSize } })
      if (res.data.code === 0) {
        this.skills = res.data.data.list
        this.total = res.data.data.total
      }
    },
    onPageChange(page) {
      this.page = page
      this.loadSkills()
    },
    editSkill(s) {
      this.editData = { id: s.id, name: s.name, description: s.description }
      this.showEdit = true
    },
    async saveEdit() {
      if (!this.editData.name || !this.editData.description) {
        this.$message.warning('请填写完整信息')
        return
      }
      const res = await api.put(`/weekly-assistant/skills/${this.editData.id}`, {
        name: this.editData.name,
        description: this.editData.description
      })
      if (res.data.code === 0) {
        this.$message.success('更新成功')
        this.showEdit = false
        this.loadSkills()
      }
    },
    async createSkill() {
      if (!this.createData.name || !this.createData.description) {
        this.$message.warning('请填写完整信息')
        return
      }
      const res = await api.post(`/weekly-assistant/skills`, this.createData)
      if (res.data.code === 0) {
        this.$message.success('添加成功')
        this.showCreate = false
        this.createData = { name: '', description: '' }
        this.loadSkills()
      }
    },
    async toggleSkill(s) {
      await api.put(`/weekly-assistant/skills/${s.id}`, { name: s.name, description: s.description, is_active: s.is_active })
    },
    async removeSkill(s) {
      try {
        await this.$confirm(`确定删除技能"${s.name}"吗？`, '确认删除', { type: 'warning' })
      } catch { return }
      const res = await api.delete(`/weekly-assistant/skills/${s.id}`)
      if (res.data.code === 0) {
        this.$message.success('已删除')
        this.loadSkills()
      }
    },
    handleClose() {
      this.$emit('update:visible', false)
    }
  }
}
</script>

<style scoped>
.skill-list {
  max-height: 400px;
  overflow-y: auto;
}
.skill-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.2s;
}
.skill-item:hover {
  background: #f8f9fa;
}
.skill-info {
  flex: 1;
}
.skill-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.skill-name {
  font-weight: 500;
  font-size: 14px;
}
.skill-desc {
  font-size: 13px;
  color: #999;
  margin-top: 2px;
}
.skill-actions {
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
.skill-pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  padding: 12px 0 0;
}
.skill-pagination-info {
  font-size: 13px;
  color: #909399;
}
</style>
