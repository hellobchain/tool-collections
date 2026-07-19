<template>
  <div class="prompt-manager">
    <el-dialog title="📝 提示词管理" :visible="visible" width="800px" :before-close="handleClose" @open="handleOpen">
      <!-- 模板列表 -->
      <div class="prompt-list">
        <div v-for="tpl in templates" :key="tpl.id" class="prompt-item" :class="{ system: tpl.is_system }">
          <div class="prompt-info">
            <div class="prompt-header">
              <span class="prompt-name">{{ tpl.name }}</span>
              <el-tag v-if="tpl.is_system" size="mini" type="info">系统</el-tag>
              <el-tag v-else size="mini" type="success">自定义</el-tag>
              <el-tag v-if="!tpl.is_active" size="mini" type="danger">已禁用</el-tag>
              <el-tag size="mini" :type="promptTypeTag(tpl.prompt_type)">{{ promptTypeLabel(tpl.prompt_type) }}</el-tag>
            </div>
            <div class="prompt-desc">{{ tpl.description || '暂无描述' }}</div>
          </div>
          <div class="prompt-actions">
            <el-button type="text" size="small" @click="viewTemplate(tpl)">查看</el-button>
            <el-button 
              v-if="!tpl.is_system" 
              type="text" 
              size="small" 
              style="color: #f56c6c;"
              @click="deleteTemplate(tpl.id)"
            >
              删除
            </el-button>
          </div>
        </div>
      </div>

      <!-- 分页 -->
      <div class="prompt-pagination" v-if="total > 0">
        <span class="prompt-pagination-info">共 {{ total }} 条</span>
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

      <!-- 创建自定义模板 -->
      <div class="create-area">
        <el-button type="primary" size="small" @click="showCreate = true">
          <i class="el-icon-plus" /> 创建自定义模板
        </el-button>
      </div>

      <!-- 查看/编辑模板弹窗 -->
      <el-dialog :title="viewTitle" :visible.sync="showView" width="700px" append-to-body>
        <el-form label-width="100px">
          <el-form-item label="模板名称">
            <el-input v-model="viewData.name" :disabled="viewData.is_system" />
          </el-form-item>
          <el-form-item label="系统提示词">
            <el-input 
              v-model="viewData.system_prompt" 
              type="textarea" 
              :rows="3" 
              :disabled="viewData.is_system"
            />
          </el-form-item>
          <el-form-item label="用户提示词">
            <el-input 
              v-model="viewData.user_prompt_template" 
              type="textarea" 
              :rows="6" 
              :disabled="viewData.is_system"
            />
            <div class="hint">
              <small>可用占位符：{fragments} {carryover} {narrative_type}</small>
            </div>
          </el-form-item>
          <el-form-item label="提示词类型" v-if="!viewData.is_system">
            <el-select v-model="viewData.prompt_type" size="small" style="width:160px">
              <el-option label="周报提示词" value="weekly" />
              <el-option label="季度提示词" value="quarterly" />
              <el-option label="年度提示词" value="yearly" />
            </el-select>
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="viewData.description" :disabled="viewData.is_system" />
          </el-form-item>
        </el-form>
        <span slot="footer">
          <el-button @click="showView = false">关闭</el-button>
              <el-button v-if="!viewData.is_system" type="primary" @click="saveTemplate">保存</el-button>
        </span>
      </el-dialog>

      <!-- 创建模板弹窗 -->
      <el-dialog title="创建自定义模板" :visible.sync="showCreate" width="700px" append-to-body>
        <el-form ref="createForm" :model="createData" label-width="100px">
          <el-form-item label="模板名称" required>
            <el-input v-model="createData.name" placeholder="如：我的老板喜欢的风格" />
          </el-form-item>
          <el-form-item label="系统提示词" required>
            <el-input 
              v-model="createData.system_prompt" 
              type="textarea" 
              :rows="3" 
            />
          </el-form-item>
          <el-form-item label="用户提示词" required>
            <el-input 
              v-model="createData.user_prompt_template" 
              type="textarea" 
              :rows="6" 
              placeholder="包含占位符：{fragments} {carryover} {narrative_type}"
            />
            <div class="hint">
              <small>💡 可用占位符：{fragments} {carryover} {narrative_type}</small>
            </div>
          </el-form-item>
          <el-form-item label="提示词类型" required>
            <el-select v-model="createData.prompt_type" size="small" style="width:160px">
              <el-option label="周报提示词" value="weekly" />
              <el-option label="季度提示词" value="quarterly" />
              <el-option label="年度提示词" value="yearly" />
            </el-select>
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="createData.description" placeholder="简单描述这个模板的适用场景" />
          </el-form-item>
        </el-form>
        <span slot="footer">
          <el-button @click="showCreate = false">取消</el-button>
          <el-button type="primary" @click="createTemplate">创建</el-button>
        </span>
      </el-dialog>
    </el-dialog>
  </div>
</template>

<script>
import promptAPI from '@/api/prompt'

export default {
  props: {
    visible: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      templates: [],
      total: 0,
      page: 1,
      pageSize: 10,
      showView: false,
      showCreate: false,
      viewTitle: '',
      viewData: {
        id: '',
        name: '',
        system_prompt: '',
        user_prompt_template: '',
        description: '',
        is_system: false,
        prompt_type: 'weekly'
      },
      createData: {
        name: '',
        system_prompt: '',
        user_prompt_template: '',
        description: '',
        prompt_type: 'weekly'
      }
    }
  },
  methods: {
    handleOpen() {
      this.page = 1
      this.loadTemplates()
    },
    promptTypeLabel(type) {
      const map = { weekly: '周报', quarterly: '季度', yearly: '年度' }
      return map[type] || '周报'
    },
    promptTypeTag(type) {
      const map = { weekly: '', quarterly: 'warning', yearly: 'success' }
      return map[type] || ''
    },
    async loadTemplates() {
      const res = await promptAPI.getTemplates({ page: this.page, page_size: this.pageSize })
      if (res.data.code === 0) {
        this.templates = res.data.data.list
        this.total = res.data.data.total
      }
    },
    onPageChange(page) {
      this.page = page
      this.loadTemplates()
    },
    selectTemplate(tpl) {
      if (!tpl.is_active) {
        this.$message.warning('该模板已禁用')
        return
      }
      this.$emit('select', tpl.id)
      this.$emit('update:visible', false)
    },
    viewTemplate(tpl) {
      this.viewData = { ...tpl }
      this.viewTitle = tpl.is_system ? `📖 ${tpl.name}（系统模板，只读）` : `✏️ ${tpl.name}`
      this.showView = true
    },
    async saveTemplate() {
      if (!this.viewData.name || !this.viewData.system_prompt || !this.viewData.user_prompt_template || !this.viewData.prompt_type) {
        this.$message.warning('请填写完整信息')
        return
      }
      try {
        await promptAPI.updateTemplate(this.viewData.id, {
          name: this.viewData.name,
          system_prompt: this.viewData.system_prompt,
          user_prompt_template: this.viewData.user_prompt_template,
          description: this.viewData.description,
          prompt_type: this.viewData.prompt_type
        })
        this.$message.success('保存成功')
        this.showView = false
        this.loadTemplates()
      } catch {
        this.$message.error('保存失败')
      }
    },
    async createTemplate() {
      if (!this.createData.name || !this.createData.system_prompt || !this.createData.user_prompt_template || !this.createData.prompt_type) {
        this.$message.warning('请填写完整信息')
        return
      }
      try {
        await promptAPI.createTemplate(this.createData)
        this.$message.success('创建成功')
        this.showCreate = false
        this.createData = { name: '', system_prompt: '', user_prompt_template: '', description: '', prompt_type: 'weekly' }
        this.loadTemplates()
      } catch {
        this.$message.error('创建失败')
      }
    },
    async deleteTemplate(id) {
      await this.$confirm('确定删除这个自定义模板吗？', '提示', { type: 'warning' })
      try {
        await promptAPI.deleteTemplate(id)
        this.$message.success('删除成功')
        this.loadTemplates()
      } catch {
        this.$message.error('删除失败')
      }
    },
    handleClose() {
      this.$emit('update:visible', false)
    }
  }
}
</script>

<style scoped>
.prompt-list {
  max-height: 400px;
  overflow-y: auto;
}
.prompt-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.2s;
}
.prompt-item:hover {
  background: #f8f9fa;
}
.prompt-item.system {
  background: #fafafa;
}
.prompt-info {
  flex: 1;
}
.prompt-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.prompt-name {
  font-weight: 500;
  font-size: 14px;
}
.prompt-desc {
  font-size: 13px;
  color: #999;
  margin-top: 2px;
}
.prompt-actions {
  display: flex;
  gap: 4px;
}
.create-area {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #eee;
  text-align: center;
}
.hint {
  margin-top: 4px;
  color: #bbb;
}
.prompt-pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 12px;
  padding: 12px 0 0;
}
.prompt-pagination-info {
  font-size: 13px;
  color: #909399;
}
</style>