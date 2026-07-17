<template>
  <div class="weekly-container">
    <!-- 顶部导航 -->
    <div class="header">
      <div class="header-left">
        <h1>📋 周报AI助手</h1>
        <el-date-picker
          v-model="selectedWeek"
          type="week"
          format="yyyy 第 WW 周"
          placeholder="选择周"
          size="small"
          class="week-selector"
          @change="onWeekChange"
        />
        <span class="week-label">{{ weekNumber }}（{{ weekStart }} ~ {{ weekEnd }}）</span>
      </div>
      <div class="header-right">
        <el-tag v-if="isFinalized" type="success">已归档</el-tag>
        <el-dropdown @command="handleUserCommand" trigger="click">
          <span class="el-dropdown-link username">
            {{ currentUser?.username }} <i class="el-icon-arrow-down el-icon--right"></i>
          </span>
          <el-dropdown-menu slot="dropdown">
            <el-dropdown-item command="profile"><i class="el-icon-setting"></i> 个人中心</el-dropdown-item>
            <el-dropdown-item command="logout" divided><i class="el-icon-switch-button"></i> 退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </el-dropdown>
      </div>
    </div>

    <!-- 上周继承提醒 -->
    <carryover-dialog 
      v-if="hasCarryover"
      :carryover="carryover"
      @confirm="handleCarryoverConfirm"
    />

    <!-- 主内容 -->
    <div class="main-content">
      <!-- 左侧：碎片收集 -->
      <div class="fragment-panel">
        <div class="panel-title">
          💡 碎片收集
          <span class="fragment-panel-title-right">
            <el-button type="text" size="mini" @click="gitlabVisible = true">Git</el-button>
            <el-button type="text" icon="el-icon-refresh" size="mini" :loading="refreshing" @click="refreshFragments" />
            <el-tag size="small">{{ fragmentCount }} 条</el-tag>
          </span>
        </div>
        <div class="fragment-input">
          <div class="fragment-input-inner">
            <el-date-picker
              v-model="fragmentDate"
              type="date"
              placeholder="选择碎片日期"
              format="yyyy-MM-dd"
              value-format="yyyy-MM-dd"
              size="mini"
              class="fragment-date-picker"
              @change="onFragmentDateChange"
            />
            <el-input
              v-model="newFragment"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 6 }"
              placeholder="记录你做了什么，比如：修复了登录超时bug"
              @keydown.enter.meta="submitFragment"
              @keydown.enter.ctrl="submitFragment"
            />
            <el-button
              type="primary"
              icon="el-icon-plus"
              circle
              size="small"
              class="fragment-submit-btn"
              @click="submitFragment"
              :disabled="!newFragment.trim() || submitting"
              :loading="submitting"
            />
          </div>
        </div>
        <div class="fragment-list" @scroll="onFragmentScroll">
          <div v-if="fragments.length === 0 && !fragmentLoading" class="empty-tip">
            <el-icon class="el-icon-edit-outline" />
            <p>还没有碎片，开始记录吧</p>
          </div>
          <div v-for="group in groupedFragments" :key="group.label" class="fragment-group">
            <div class="fragment-group-title">{{ group.year }}年{{ group.label }}</div>
            <div v-for="(f, idx) in group.items" :key="f.id" class="fragment-item">
              <span class="fragment-index">{{ idx + 1 }}</span>
              <div class="fragment-body">
                <span class="fragment-date" v-if="f.date">{{ f.date }}</span>
                <el-tag v-if="f.is_carried" size="mini" type="warning" class="carryover-tag">遗留</el-tag>
                <span class="fragment-content" v-html="formatContent(f.content)"></span>
              </div>
              <el-button type="text" icon="el-icon-close" class="delete-btn" @click="removeFragment(f.id)" />
            </div>
          </div>
          <div v-if="fragmentLoading" class="scroll-loading">加载中...</div>
          <div v-if="!hasMoreFragments && fragments.length > 0" class="scroll-end">— 没有更多了 —</div>
        </div>
        <div class="fragment-hint">
          <small>💡 每天记录几件小事，周五自动生成周报</small>
        </div>
      </div>
      <!-- 中间：草稿区 -->
      <div class="draft-panel">
        <div class="panel-title">
          📌 周报草稿
        </div>
        <!-- 顶部栏：叙事类型 + 管理 -->
        <div class="draft-topbar">
          <el-button-group size="mini">
            <el-button :type="narrativeType === '攻坚' ? 'primary' : ''" size="mini" @click="switchNarrative('攻坚')">攻坚</el-button>
            <el-button :type="narrativeType === '协作' ? 'primary' : ''" size="mini" @click="switchNarrative('协作')">协作</el-button>
            <el-button :type="narrativeType === '稳健' ? 'primary' : ''" size="mini" @click="switchNarrative('稳健')">稳健</el-button>
          </el-button-group>
          <div class="draft-topbar-right">
            <span class="topbar-label">提示词风格</span>
            <el-select 
              v-model="selectedPromptId" 
              placeholder="风格" 
              size="mini" 
              style="width: 100px;"
              clearable
              filterable
              @change="onPromptChange"
              @visible-change="onPromptDropdownOpen"
            >
              <el-option 
                v-for="tpl in promptTemplates" 
                :key="tpl.id" 
                :label="tpl.name" 
                :value="tpl.id"
                :disabled="!tpl.is_active"
              />
            </el-select>
            <el-button type="text" icon="el-icon-setting" @click="showPromptManager = true">提示词</el-button>
            <el-button type="text" icon="el-icon-user" @click="showSkillManager = true">技能</el-button>
            <el-button type="text" icon="el-icon-link" @click="showGitProjectManager = true">Git</el-button>
          </div>
        </div>

        <!-- 提示词管理弹窗 -->
        <prompt-manager 
          :visible.sync="showPromptManager" 
          @select="onPromptSelected"
        />

        <!-- 技能管理弹窗 -->
        <skill-manager :visible.sync="showSkillManager" />

        <!-- Git项目管理弹窗 -->
        <git-project-manager :visible.sync="showGitProjectManager" />

        <!-- 内容展示区 -->
        <div class="draft-display-area">
          <el-tabs v-model="draftMode">
            <el-tab-pane label="编辑" name="edit">
              <el-input
                :value="draftContent"
                type="textarea"
                :rows="20"
                :placeholder="draftPlaceholder"
                :disabled="isFinalized"
                @input="updateDraft"
              />
            </el-tab-pane>
            <el-tab-pane label="预览" name="preview">
              <div class="draft-preview-content" v-if="draftContent" v-html="renderedDraft"></div>
              <div class="draft-preview-empty" v-else>
                <span v-if="isGenerating || isStreaming">AI生成中...</span>
                <span v-else>暂无内容，请先生成草稿</span>
              </div>
            </el-tab-pane>
          </el-tabs>
        </div>

        <div class="draft-actions-bottom">
          <div class="draft-bottom-left">
            <span class="stream-toggle">
              <el-switch v-model="streamEnabled" active-color="#4F6EF7" />
              <span class="stream-label">流式生成</span>
            </span>
            <el-button 
              type="primary" 
              :disabled="!canGenerate || isGenerating || isStreaming"
              @click="handleGenerate"
              class="generate-btn"
            >
              {{ btnText }}
            </el-button>
          </div>
          <div class="draft-bottom-right">
            <el-button type="success" :disabled="!draftContent || isFinalized || finalizing" :loading="finalizing" @click="handleFinalize">
              定稿归档
            </el-button>
            <span v-if="isFinalized" class="finalized-tip">✅ 本周已归档</span>
          </div>
        </div>
      </div>

      <!-- 右侧：三栏 5:3:2，各自独立滚动 -->
      <div class="history-panel">
        <div class="panel-title">
          ⏳ 历史汇总
        </div>
        <div class="panel-section section-weekly">
          <div class="section-header">
            📚 周报汇总
            <span class="section-header-right">
              <el-button type="text" icon="el-icon-download" size="mini" :loading="weekExporting" @click="handleWeekExport" />
              <el-button type="text" icon="el-icon-refresh" size="mini" :loading="historyRefreshLoading" @click="loadHistory(true)" />
            </span>
          </div>
          <div class="history-search">
            <el-date-picker
              v-model="historyDateRange"
              type="daterange"
              range-separator="~"
              start-placeholder="周报开始日期"
              end-placeholder="周报结束日期"
              format="yyyy-MM-dd"
              value-format="yyyy-MM-dd"
              size="small"
              @change="loadHistory()"
            />
          </div>
          <div class="section-scroll" v-loading="historyLoading" element-loading-text="加载中..." @scroll="onHistoryScroll">
            <div v-if="history.length === 0" class="empty-tip"><p>暂无历史记录</p></div>
            <div v-for="group in groupedHistory" :key="group.label" class="history-group">
              <div class="history-group-title">{{ group.year }}年{{ group.label }}</div>
              <div v-for="(item, idx) in group.items" :key="idx" class="history-item">
                <div class="history-item-main" @click="previewHistory(item)">
                  <div class="history-item-header">
                    <span class="history-week-label">{{ item.week_start }}</span>
                    <el-tag size="mini">{{ item.narrative_type }}</el-tag>
                  </div>
                  <div class="history-item-preview" v-html="formatContent(truncate(item.content, 120))"></div>
                </div>
                <el-button type="text" icon="el-icon-close" class="history-delete-btn" :disabled="deletingReport === item.id" @click.stop="handleDeleteReport(item)" />
              </div>
            </div>
            <div v-if="!hasMoreHistory && history.length > 0" class="scroll-end">— 没有更多了 —</div>
          </div>
        </div>

        <div class="panel-section section-quarter">
          <div class="section-header">
            📊 季度汇总
            <span class="section-header-right">
              <el-button type="text" icon="el-icon-download" size="mini" :loading="this.getSummaryExporting('quarter')" @click="handleSummaryExport('quarter')" />
              <el-button type="text" icon="el-icon-refresh" size="mini" :loading="summaryQuarterRefreshLoading" @click="loadQuarterSummaries()" />
            </span>
          </div>
          <div class="section-generate">
            <el-select v-model="genQYear" size="mini" style="width:80px" filterable>
              <el-option v-for="y in yearOptions" :key="y.value" :label="y.label" :value="y.value" />
            </el-select>
            <el-select v-model="genQQ" size="mini" style="width:70px" filterable>
              <el-option v-for="y in quarterOptions" :key="y.value" :label="y.label" :value="y.value" />
            </el-select>
            <el-select v-model="genQPromptId" size="mini" style="width:110px" placeholder="提示风格" filterable clearable :loading="genQPromptLoading" @visible-change="(v) => v && loadPromptsByType('quarterly', 'genQPrompt')">
              <el-option v-for="t in genQPromptOptions" :key="t.id" :label="t.name" :value="t.id" />
            </el-select>
            <el-button type="primary" size="mini" :loading="genQLoading" @click="generateSummary('quarter', genQYear + '-' + genQQ)">生成</el-button>
          </div>
          <div class="section-scroll">
            <div class="summary-list-header">
              <span>季度</span>
              <span>创建时间</span>
            </div>
            <div v-if="quarterSummaries.length === 0" class="empty-tip" style="padding:6px 0;"><p style="font-size:12px;">暂无</p></div>
            <div v-for="s in quarterSummaries" :key="s.id" class="summary-item">
              <div class="summary-item-main" @click="showSummaryDetail(s)">
                <span class="summary-item-label">{{ s.period_value }}</span>
                <span class="summary-item-time">{{ s.created_at?.slice(0,19)?.replace('T', ' ') }}</span>
              </div>
              <el-button type="text" icon="el-icon-close" class="summary-delete-btn" @click.stop="deleteSummary(s)" />
            </div>
          </div>
        </div>

        <div class="panel-section section-year">
          <div class="section-header">
            📅 年度汇总
            <span class="section-header-right">
              <el-button type="text" icon="el-icon-download" size="mini" :loading="this.getSummaryExporting('year')" @click="handleSummaryExport('year')" />
              <el-button type="text" icon="el-icon-refresh" size="mini" :loading="summaryYearRefreshLoading" @click="loadYearSummaries()" />
            </span>
          </div>
          <div class="section-generate">
            <el-select v-model="genYValue" size="mini" style="width:110px" filterable>
              <el-option v-for="o in yearOptions" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
            <el-select v-model="genYPromptId" size="mini" style="width:110px" placeholder="提示风格" filterable clearable :loading="genYPromptLoading" @visible-change="(v) => v && loadPromptsByType('yearly', 'genYPrompt')">
              <el-option v-for="t in genYPromptOptions" :key="t.id" :label="t.name" :value="t.id" />
            </el-select>
            <el-button type="primary" size="mini" :loading="genYLoading" @click="generateSummary('year', genYValue)">生成</el-button>
          </div>
          <div class="section-scroll">
            <div class="summary-list-header">
              <span>年度</span>
              <span>创建时间</span>
            </div>
            <div v-if="yearSummaries.length === 0" class="empty-tip" style="padding:6px 0;"><p style="font-size:12px;">暂无</p></div>
            <div v-for="s in yearSummaries" :key="s.id" class="summary-item">
              <div class="summary-item-main" @click="showSummaryDetail(s)">
                <span class="summary-item-label">{{ s.period_value }}年</span>
                <span class="summary-item-time">{{ s.created_at?.slice(0,19)?.replace('T', ' ') }}</span>
              </div>
              <el-button type="text" icon="el-icon-close" class="summary-delete-btn" @click.stop="deleteSummary(s)" />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- GitLab 配置 -->
    <el-dialog title="从 GitLab 导入提交记录" :visible.sync="gitlabVisible" width="520px" @open="loadGitProjects">
      <el-form label-width="100px" size="small">
        <el-form-item label="选择项目">
          <el-select
            v-model="selectedGitProjectId"
            placeholder="搜索 Git 项目...（选中后自动回填下方信息）"
            filterable
            clearable
            remote
            :remote-method="searchGitProjects"
            :loading="gitProjectSearching"
            style="width:100%"
            @change="onGitProjectSelected"
          >
            <el-option
              v-for="p in gitProjectOptions"
              :key="p.id"
              :label="p.project_name"
              :value="p.id"
            >
              <span>{{ p.project_name }}</span>
              <span style="float:right;color:#999;font-size:12px">{{ p.base_url }}</span>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="GitLab地址" required>
            <el-input v-model="gitlabForm.base_url" name="gitlab_url" placeholder="https://gitlab.company.com" />
        </el-form-item>
        <el-form-item label="Token" required>
          <el-input v-model="gitlabForm.token" name="gitlab_token" type="password" placeholder="GitLab Personal Access Token" show-password />
        </el-form-item>
        <el-form-item label="项目ID" required>
          <el-input v-model="gitlabForm.project_id" name="gitlab_project_id" placeholder="项目ID（数字）" />
        </el-form-item>
        <el-form-item label="项目名称" required>
          <el-input v-model="gitlabForm.project_name" name="gitlab_project_name" placeholder="项目名称，用于碎片标识" />
        </el-form-item>
        <el-form-item label="项目分支" required>
          <el-input v-model="gitlabForm.project_branch" name="gitlab_branch" placeholder="项目分支" />
        </el-form-item>
        <el-form-item label="提交名称">
          <el-input v-model="gitlabForm.email" name="gitlab_email" placeholder="提交时使用的名称，用于过滤" />
        </el-form-item>
        <el-form-item label="日期范围" required>
          <el-date-picker
            v-model="gitlabForm.dateRange"
            type="daterange"
            range-separator="~"
            start-placeholder="提交开始日期"
            end-placeholder="提交结束日期"
            format="yyyy-MM-dd"
            value-format="yyyy-MM-dd"
            style="width:100%"
          />
        </el-form-item>
      </el-form>
      <div v-if="gitlabCommits.length > 0" class="gitlab-commits-list">
        <div class="gitlab-commits-header">
          <span>共 {{ gitlabCommits.length }} 条提交</span>
          <el-button type="primary" size="mini" :loading="gitlabImporting" @click="importGitLabCommits">全部添加为碎片</el-button>
        </div>
        <div v-for="(c, i) in gitlabCommits" :key="i" class="gitlab-commit-item">
          <span class="gitlab-commit-msg">{{ c.message }}</span>
          <span class="gitlab-commit-date">{{ c.authored_date?.slice(0, 10) }}</span>
        </div>
      </div>
      <span slot="footer">
        <el-button @click="gitlabVisible = false">取消</el-button>
        <el-button type="primary" :loading="gitlabLoading" @click="fetchGitLabCommits">拉取提交</el-button>
      </span>
    </el-dialog>

    <!-- 个人中心 -->
    <el-dialog title="个人中心" :visible.sync="profileVisible" width="400px">
      <el-form label-width="100px">
        <el-form-item label="用户名">
          <el-input :value="currentUser?.username" disabled />
        </el-form-item>
        <el-form-item label="原密码">
          <el-input v-model="pwdForm.oldPassword" name="old_password" placeholder="请输入原密码" :type="pwdShow.old ? 'text' : 'password'">
            <i slot="suffix" class="el-icon-view" style="cursor:pointer;font-size:16px" @click="pwdShow.old = !pwdShow.old" />
          </el-input>
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="pwdForm.newPassword" name="new_password" placeholder="请输入新密码（至少6位）" :type="pwdShow.new ? 'text' : 'password'">
            <i slot="suffix" class="el-icon-view" style="cursor:pointer;font-size:16px" @click="pwdShow.new = !pwdShow.new" />
          </el-input>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button @click="profileVisible = false">取消</el-button>
        <el-button type="primary" :loading="pwdLoading" @click="handleChangePassword">修改密码</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
// 导入提示词API和组件
import promptAPI from '@/api/prompt'
import PromptManager from '@/components/PromptManager.vue'
import SkillManager from '@/components/SkillManager.vue'
import GitProjectManager from '@/components/GitProjectManager.vue'
import { marked } from 'marked'
import { mapState, mapGetters, mapActions } from 'vuex'
import api from '@/api'
import {
  NARRATIVE_TYPES, DEFAULT_NARRATIVE, HISTORY_PREVIEW_TRUNCATE,
  TOAST_DURATION, STORAGE_TOKEN, AUTH_SCHEME, SUCCESS_CODE,
  DEFAULT_GIT_BRANCH, PASSWORD_MIN_LENGTH, YEAR_RANGE,
  EXPORT_WEEKLY_FILENAME, EXPORT_QUARTER_FILENAME, EXPORT_YEAR_FILENAME,
  TEMPLATE_PAGE_SIZE, GIT_PROJECT_PAGE_SIZE
} from '@/constants'
import CarryoverDialog from '@/components/CarryoverDialog.vue'

function copyToClipboard(text) {
  navigator.clipboard.writeText(text).catch(() => {
    const el = document.createElement('textarea')
    el.value = text
    el.style.position = 'fixed'
    el.style.top = '9999px'
    el.style.left = '9999px'
    document.body.appendChild(el)
    el.select()
    document.execCommand('copy')
    document.body.removeChild(el)
  })
}
function showCopySuccess() {
  const div = document.createElement('div')
  div.style.cssText = 'position:fixed;top:20px;left:50%;transform:translateX(-50%);background:#67c23a;color:#fff;padding:10px 20px;border-radius:4px;z-index:999999;font-size:14px;'
  div.textContent = '已复制'
  document.body.appendChild(div)
  setTimeout(() => document.body.removeChild(div), TOAST_DURATION)
}

export default {
  components: { CarryoverDialog, PromptManager, SkillManager, GitProjectManager },
  data() {
    return {
      promptTemplates: [],        // 提示词模板列表
      selectedPromptId: '',       // 当前选中的模板ID
      showPromptManager: false,
      showSkillManager: false,
      showGitProjectManager: false,
      workInput: '',
      streamEnabled: true,
      newFragment: '',
      fragmentDate: '',
      selectedWeek: '',
      historyDateRange: null,
      profileVisible: false,
      pwdLoading: false,
      pwdForm: { oldPassword: '', newPassword: '' },
      pwdShow: { old: false, new: false },
      draftMode: 'edit',
      refreshing: false,
      submitting: false,
      finalizing: false,
      weekExporting: false,
      summaryQuarterExporting: false,
      summaryYearExporting: false,
      historyRefreshLoading: false,
      deletingReport: false,
      genQYear: '',
      genQQ: 'Q1',
      genYValue: '',
      genQLoading: false,
      genYLoading: false,
      genQPromptId: '',
      genYPromptId: '',
      genQPromptOptions: [],
      genYPromptOptions: [],
      genQPromptLoading: false,
      genYPromptLoading: false,
      summaryQuarterRefreshLoading: false,
      summaryYearRefreshLoading: false,
      summaryRefreshLoading: false,
      summaryYearRefreshLoading: false,
      summaryQuarterRefreshLoading: false,
      quarterSummaries: [],
      yearSummaries: [],
      gitlabVisible: false,
      gitlabLoading: false,
      gitlabImporting: false,
      gitlabForm: {
        base_url: '',
        token: '',
        project_id: '',
        project_name: '',
        project_branch: '',
        email: '',
        dateRange: null
      },
      gitlabCommits: [],
      selectedGitProjectId: '',
      gitProjectOptions: [],
      gitProjectSearching: false,
      gitProjectPage: 1,
      gitProjectKeyword: ''
    }
  },
  computed: {
    quarterOptions() {
      const opts = []
      for (let i = 1; i <= 4; i++) opts.push({ label: `Q${i}`, value: `Q${i}` })
      if (!this.genQQ) this.genQQ = opts[0].value
      return opts
    },
    yearOptions() {
      const y = new Date().getFullYear()
      const opts = []
      for (let i = y; i >= y - 10; i--) opts.push({ label: `${i}年`, value: `${i}` })
      if (!this.genYValue) this.genYValue = opts[0].value
      if (!this.genQYear) this.genQYear = opts[0].value
      return opts
    },
    ...mapState('weekly', [
      'weekStart', 'weekEnd', 'weekNumber', 'fragments', 'draftContent', 'narrativeType',
      'carryover', 'carryoverConfirmed', 'isGenerating', 'isStreaming', 'isFinalized',
      'history', 'historyLoading', 'fragmentLoading', 'nextWeekPlan'
    ]),
    ...mapGetters('auth', ['currentUser']),
    ...mapGetters('weekly', ['fragmentCount', 'hasCarryover', 'canGenerate', 'hasMoreFragments', 'hasMoreHistory']),
    groupedFragments() {
      const groups = {}
      for (const f of this.fragments) {
        const d = f.occurred_at ? new Date(f.occurred_at) : (this.weekStart ? new Date(this.weekStart) : new Date())
        const year = d.getFullYear()
        const month = d.getMonth() + 1
        const week = this.getISOWeek(d)
        const key = `${year}-${String(month).padStart(2, '0')}-W${String(week).padStart(2, '0')}`
        if (!groups[key]) groups[key] = { key, year, month, week, label: `${month}月 - 第${week}周`, items: [] }
        groups[key].items.push(f)
      }
      return Object.values(groups).sort((a, b) => a.key < b.key ? 1 : -1)
    },
    renderedDraft() {
      return marked(this.draftContent || '', { breaks: true })
    },
    draftPlaceholder() {
      if (this.isGenerating || this.isStreaming) return 'AI生成中...'
      return '暂无内容，请先生成草稿'
    },
    btnText() {
      if (this.isGenerating) return '生成中...'
      if (this.isStreaming) return '流式生成中...'
      return this.draftContent ? '再次生成' : '生成草稿'
    },
    groupedHistory() {
      const groups = {}
      for (const item of this.history) {
        const d = new Date(item.week_start)
        const year = d.getFullYear()
        const month = d.getMonth() + 1
        const week = this.getISOWeek(d)
        const key = `${year}-${String(month).padStart(2, '0')}-W${String(week).padStart(2, '0')}`
        if (!groups[key]) groups[key] = { key, year, month, week, label: `${month}月 - 第${week}周`, items: [] }
        groups[key].items.push(item)
      }
      return Object.values(groups).sort((a, b) => a.key < b.key ? 1 : -1)
    }
  },
  mounted() {
    this.initWeek()
    this.loadHistory()
    this.loadPromptTemplates()
    this.loadSummaries()
    // 同步周一到日期选择器
    this.$watch('weekStart', (val) => {
      if (val && !this.selectedWeek) this.selectedWeek = new Date(val)
    })
  },
  methods: {
    ...mapActions('weekly', [
      'initWeek', 'addFragment', 'deleteFragment', 
      'generateDraft', 'finalize', 'confirmCarryover', 'fetchHistory', 'generateDraftStream', 'deleteReport',
      'loadFragments'
    ]),
    async loadPromptTemplates(promptType = 'weekly') {
      const res = await promptAPI.getTemplates({ page: 1, page_size: 100, prompt_type: promptType})
      if (res.data.code === SUCCESS_CODE) {
        const list = res.data.data.list || res.data.data
        this.promptTemplates = list.filter(t => t.is_active)
        // 默认选中第一个
        if (this.promptTemplates.length > 0 && !this.selectedPromptId) {
          this.selectedPromptId = this.promptTemplates[0].id
        }
      }
    },
    onPromptDropdownOpen(visible) {
      if (visible) this.loadPromptTemplates()
    },
    onPromptChange() {
    },
    onPromptSelected(templateId) {
      this.selectedPromptId = templateId
    },
    async loadPromptsByType(promptType, target) {
      const loadingKey = target === 'genQPrompt' ? 'genQPromptLoading' : 'genYPromptLoading'
      this[loadingKey] = true
      try {
        const res = await promptAPI.getTemplates({ page: 1, page_size: 100, prompt_type: promptType })
        if (res.data.code === 0) {
          const list = res.data.data.list || res.data.data
          const optKey = target === 'genQPrompt' ? 'genQPromptOptions' : 'genYPromptOptions'
          this[optKey] = list.filter(t => t.is_active)
        }
      } finally {
        this[loadingKey] = false
      }
    },
    onFragmentScroll(e) {
      const el = e.target
      if (el.scrollTop + el.clientHeight >= el.scrollHeight - 10 && this.hasMoreFragments && !this.fragmentLoading) {
        this.loadFragments({ page: this.$store.state.weekly.fragmentPage + 1, append: true })
      }
    },
    onHistoryScroll(e) {
      const el = e.target
      if (el.scrollTop + el.clientHeight >= el.scrollHeight - 10 && this.hasMoreHistory) {
        this.$store.dispatch('weekly/fetchHistory', {
          page: this.$store.state.weekly.historyPage + 1,
          append: true,
          week_start: this.historyDateRange ? this.historyDateRange[0] : undefined,
          week_end: this.historyDateRange ? this.historyDateRange[1] : undefined
        })
      }
    },
    getISOWeek(d) {
      const date = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()))
      date.setUTCDate(date.getUTCDate() + 4 - (date.getUTCDay() || 7))
      const yearStart = new Date(Date.UTC(date.getUTCFullYear(), 0, 1))
      return Math.ceil((((date - yearStart) / 86400000) + 1) / 7)
    },
    formatContent(text) {
      return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/\n/g, '<br>')
    },
    truncate(text, len) {
      if (!text || text.length <= len) return text
      return text.slice(0, len) + '...'
    },
    previewHistory(item) {
      const h = this.$createElement
      const content = item.content
      const formatted = marked.parse(content, { breaks: true })
      this.$msgbox({
        title: `周报 - ${item.week_start}（${item.narrative_type}）`,
        message: h(          'div', {
            style: 'position: relative; max-height: 70vh; overflow-y: auto;',
          }, [
          h('button', {
            style: 'position:absolute;top:0;right:0;z-index:1;border:none;background:#ecf5ff;color:#409eff;padding:4px 8px;border-radius:4px;cursor:pointer;font-size:12px;',
            on: {
              click: function () {
                copyToClipboard(content)
                showCopySuccess()
              }
            }
          }, '复制'),
          h('div', {
            style: 'line-height: 1.8; font-size: 14px; padding: 36px 16px 16px;',
            domProps: { innerHTML: formatted }
          })
        ]),
        dangerouslyUseHTMLString: false,
        showCancelButton: false,
        confirmButtonText: '关闭',
        width: '60%'
      }).catch(() => {})
    },
    async loadHistory(showTip) {
      const params = { page: 1 }
      if (this.historyDateRange) {
        params.week_start = this.historyDateRange[0]
        params.week_end = this.historyDateRange[1]
      }
      this.$store.dispatch('weekly/fetchHistory', params)
      if (showTip) this.$message.success('已刷新周报汇总')
    },
    async refreshFragments() {
      this.refreshing = true
      try {
        const ok = await this.initWeek()
        if (ok) {
          this.loadFragments({ page: 1, append: false })
          this.$message.success('已刷新碎片')
        }
      } finally {
        this.refreshing = false
      }
    },
    async onWeekChange(val) {
      if (!val) return
      // val is a Date (the Monday of the selected week)
      const year = val.getFullYear()
      const month = String(val.getMonth() + 1).padStart(2, '0')
      const day = String(val.getDate()).padStart(2, '0')
      const weekStart = `${year}-${month}-${day}`
      await this.$store.dispatch('weekly/switchWeek', weekStart)
      this.selectedWeek = val
    },
    async submitFragment() {
      if (this.submitting) return
      const content = this.newFragment.trim()
      if (!content) return
      this.submitting = true
      try {
        const ok = await this.addFragment({ content, date: this.fragmentDate || undefined })
        if (ok) this.newFragment = ''
      } finally {
        this.submitting = false
      }
    },
    onFragmentDateChange(val) {
      // just store the value, submitFragment will use it
    },
    async removeFragment(id) {
      this.refreshing = true
      try { await this.deleteFragment(id) }
      finally { this.refreshing = false }
    },
    switchNarrative(type) {
      this.$store.commit('weekly/SET_NARRATIVE', type)
    },
    handleGenerate() {
      const payload = { template_id: this.selectedPromptId || undefined }
      if (this.streamEnabled) {
        this.$store.dispatch('weekly/generateDraftStream', payload)
      } else {
        this.$store.dispatch('weekly/generateDraft', payload)
      }
    },
    updateDraft(val) {
      this.$store.commit('weekly/SET_DRAFT', val)
    },
    async handleCarryoverConfirm({ kept, dropped }) {
      const ok = await this.confirmCarryover({ keptIds: kept, droppedIds: dropped })
      if (ok) this.$message.success('已确认上周遗留事项')
    },
    async loadSummaries() {
      this.summaryRefreshLoading = true
      try {
        const res = await api.get('/api/week/summaries', { params: {} })
        if (res.data.code === SUCCESS_CODE) {
          this.quarterSummaries = res.data.data.quarter || []
          this.yearSummaries = res.data.data.year || []
        }
      } finally {
        this.summaryRefreshLoading = false
      }
    },
    async loadQuarterSummaries() {
      this.summaryQuarterRefreshLoading = true
      try {
        const res = await api.get('/api/week/summaries', { params: { type: 'quarter' } })
        if (res.data.code === SUCCESS_CODE) {
          this.quarterSummaries = res.data.data.quarter || []
          this.$message.success('已刷新季度汇总')
        }
      } finally {
        this.summaryQuarterRefreshLoading = false
      }
    },
    async loadYearSummaries() {
      this.summaryYearRefreshLoading = true
      try {
        const res = await api.get('/api/week/summaries', { params: { type: 'year' } })
        if (res.data.code === SUCCESS_CODE) {
          this.yearSummaries = res.data.data.year || []
          this.$message.success('已刷新年度汇总')
        }
      } finally {
        this.summaryYearRefreshLoading = false
      }
    },
    async generateSummary(type, value) {
      if (!value) { this.$message.warning('请选择周期'); return }
      const loadingKey = type === 'quarter' ? 'genQLoading' : 'genYLoading'
      this[loadingKey] = true
      try {
        const res = await api.get('/api/week/summary', { params: { type, value } })
        if (res.data.code === SUCCESS_CODE) {
          this.$message.success('季度/年度汇总已生成并保存')
          this.loadQuarterSummaries()
          this.loadYearSummaries()
        }
      } finally {
        this[loadingKey] = false
      }
    },
    async deleteSummary(s) {
      try {
        await this.$confirm(`确定删除 ${this.formatPeriodLabel(s)} 的汇总报告吗？`, '确认删除', {
          confirmButtonText: '确定删除',
          cancelButtonText: '取消',
          type: 'warning'
        })
      } catch { return }
      try {
        const res = await api.delete(`/api/week/summary/${s.id}`)
        if (res.data.code === SUCCESS_CODE) {
          this.$message.success('已删除')
          if (s.period_type === 'quarter') {
            this.loadQuarterSummaries()
          } else if (s.period_type === 'year'){
            this.loadYearSummaries()
          }
        }
      } catch {}
    },
    formatPeriodLabel(s) {
      if (s.period_type === 'year') return s.period_value + '年'
      const v = s.period_value
      return v.slice(0, 4) + '年' + v.slice(5)
    },
    showSummaryDetail(s) {
      const h = this.$createElement
      const content = s.content
      const formatted = marked.parse(content, { breaks: true })
      this.$msgbox({
        title: `📊 ${this.formatPeriodLabel(s)}工作总结`,
        message: h('div', {
          style: 'position: relative; max-height: 70vh; overflow-y: auto;',
        }, [
          h('button', {
            style: 'position:absolute;top:0;right:0;z-index:1;border:none;background:#ecf5ff;color:#409eff;padding:4px 8px;border-radius:4px;cursor:pointer;font-size:12px;',
            on: {
              click: function () {
                copyToClipboard(content)
                showCopySuccess()
              }
            }
          }, '复制'),
          h('div', {
            style: 'line-height: 1.8; font-size: 14px; padding: 36px 16px 16px;',
            domProps: { innerHTML: formatted }
          })
        ]),
        dangerouslyUseHTMLString: false,
        showCancelButton: false,
        confirmButtonText: '关闭',
        width: '65%'
      }).catch(() => {})
    },
    async handleFinalize() {
      if (this.finalizing) return
      try {
        const now = new Date()
        const mon = new Date(now)
        mon.setDate(mon.getDate() - ((now.getDay() + 6) % 7))
        const currentMon = `${mon.getFullYear()}-${String(mon.getMonth() + 1).padStart(2, '0')}-${String(mon.getDate()).padStart(2, '0')}`
        const isCurrentWeek = !this.$store.state.weekly.currentWeekStart || this.$store.state.weekly.currentWeekStart === currentMon
        const weekLabel = isCurrentWeek ? '本周' : `第${this.weekNumber}周`
        await this.$confirm(`定稿后${weekLabel}周报将归档，确定提交吗？`, '确认归档', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        })
      } catch { return }
      this.finalizing = true
      try {
        const ok = await this.finalize()
        if (ok) {
          this.$message.success('✅ 周报已归档！')
          this.loadHistory(true)
        }
      } finally {
        this.finalizing = false
      }
    },
    async handleDeleteReport(item) {
      if (this.deletingReport) return
      try {
        await this.$confirm(`确定删除 ${item.week_start} 的周报吗？删除后不可恢复。`, '确认删除', {
          confirmButtonText: '确定删除',
          cancelButtonText: '取消',
          type: 'warning'
        })
      } catch { return }
      this.deletingReport = true
      try {
        const ok = await this.deleteReport(item.id)
        if (ok) {
          this.$message.success('已删除')
          this.loadHistory()
          this.initWeek()
        }
      } finally {
        this.deletingReport = false
      }
    },
    handleWeekExport() {
      if (this.weekExporting) return
      this.weekExporting = true
      const token = localStorage.getItem(STORAGE_TOKEN)
      const params = new URLSearchParams()
      if (this.historyDateRange) {
        params.set('week_start', this.historyDateRange[0])
        params.set('week_end', this.historyDateRange[1])
      }
      const url = `/api/week/history/weeks/export?${params.toString()}`
      const xhr = new XMLHttpRequest()
      xhr.open('GET', url)
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
      xhr.responseType = 'blob'
      xhr.onload = () => {
        this.weekExporting = false
        if (xhr.status === 200) {
          const blob = xhr.response
          const link = document.createElement('a')
          link.href = URL.createObjectURL(blob)
          link.download = 'weekly_reports.xlsx'
          link.click()
          URL.revokeObjectURL(link.href)
        } else {
          this.$message.error(xhr.statusText)
        }
      }
      xhr.onerror = () => { this.weekExporting = false }
      xhr.send()
    },
    setSummaryExporting(summaryType,statusValue) {
      if (summaryType == 'quarter') {
        this.summaryQuarterExporting = statusValue
      } else {
        this.summaryYearExporting = statusValue
      }
    },
    getSummaryExporting(summaryType) {
      if (summaryType == 'quarter') {
        return this.summaryQuarterExporting
      } else {
        return  this.summaryYearExporting
      }
    },
    handleSummaryExport(summaryType) {
      if (this.getSummaryExporting(summaryType)) return
      this.setSummaryExporting(summaryType,true)
      const token = localStorage.getItem(STORAGE_TOKEN)
      const params = new URLSearchParams()
      if (summaryType) {
        params.set('type', summaryType)
      }
      const url = `/api/week/history/summaries/export?${params.toString()}`
      const xhr = new XMLHttpRequest()
      xhr.open('GET', url)
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
      xhr.responseType = 'blob'
      xhr.onload = () => {
        this.setSummaryExporting(summaryType,false)
        if (xhr.status === 200) {
          const blob = xhr.response
          const link = document.createElement('a')
          link.href = URL.createObjectURL(blob)
          if (summaryType == "quarter") {
            link.download = 'quarter_history.xlsx'
          } else {
            link.download = 'year_history.xlsx'
          }
          link.click()
          URL.revokeObjectURL(link.href)
        } else {
          this.$message.error(xhr.statusText)
        }
      }
      xhr.onerror = () => { this.setSummaryExporting(summaryType,false) }
      xhr.send()
    },
    async fetchGitLabCommits() {
      const f = this.gitlabForm
      if (!f.base_url || !f.token || !f.project_id || !f.project_branch || !f.project_name || !f.dateRange) {
        this.$message.warning('请填写完整配置信息')
        return
      }
      this.gitlabLoading = true
      this.gitlabCommits = []
      try {
        const res = await api.post('/api/gitlab/commits', {
          base_url: f.base_url,
          token: f.token,
          project_id: f.project_id,
          project_name: f.project_name,
          branch: f.project_branch,
          email: f.email || '',
          start_date: f.dateRange[0],
          end_date: f.dateRange[1]
        })
        if (res.data.code === SUCCESS_CODE) {
          this.gitlabCommits = res.data.data.commits || []
          if (this.gitlabCommits.length === 0) this.$message.info('没有找到提交记录')
          else this.$message.success(`拉取到 ${this.gitlabCommits.length} 条提交`)
        }
      } catch {} finally {
        this.gitlabLoading = false
      }
    },
    async loadGitProjects() {
      this.gitProjectKeyword = ''
      this.gitProjectPage = 1
      const res = await api.get('/api/git-projects', { params: { page: 1, page_size: 50 } })
      if (res.data.code === SUCCESS_CODE) this.gitProjectOptions = res.data.data.list || []
    },
    async searchGitProjects(keyword) {
      this.gitProjectKeyword = keyword
      this.gitProjectSearching = true
      try {
        const res = await api.get('/api/git-projects', { params: { page: 1, page_size: 50, keyword } })
        if (res.data.code === SUCCESS_CODE) this.gitProjectOptions = res.data.data.list || []
      } finally {
        this.gitProjectSearching = false
      }
    },
    onGitProjectSelected(id) {
      if (!id) return
      const p = this.gitProjectOptions.find(x => x.id === id)
      if (p) {
        this.gitlabForm.base_url = p.base_url
        this.gitlabForm.token = p.token
        this.gitlabForm.project_id = p.project_id
        this.gitlabForm.project_name = p.project_name
        this.gitlabForm.project_branch = p.branch || 'master'
      }
    },
    async importGitLabCommits() {
      if (this.gitlabCommits.length === 0) return
      this.gitlabImporting = true
      let count = 0
      for (const c of this.gitlabCommits) {
        try {
          const date = c.authored_date ? c.authored_date.slice(0, 10) : undefined
          const projectTag = c.project_name ? `[${c.project_name}]` : ''
          const ok = await this.addFragment({
            content: `[Git] ${projectTag} ${c.message}`.trim(),
            date
          })
          if (ok) count++
        } catch {}
      }
      this.gitlabImporting = false
      this.$message.success(`已添加 ${count} 条提交记录到碎片`)
      if (count > 0) {
        this.gitlabCommits = []
        this.gitlabVisible = false
        this.refreshFragments()
      }
    },
    handleUserCommand(cmd) {
      if (cmd === 'profile') {
        this.pwdForm = { oldPassword: '', newPassword: '' }
        this.profileVisible = true
      } else if (cmd === 'logout') {
        this.$confirm('确定要退出登录吗？', '退出确认', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }).then(() => {
          this.$store.dispatch('auth/logout')
          this.$router.push('/login')
        }).catch(() => {})
      }
    },
    async handleChangePassword() {
      if (!this.pwdForm.oldPassword || !this.pwdForm.newPassword) {
        this.$message.warning('请填写完整信息')
        return
      }
      if (this.pwdForm.newPassword.length < 6) {
        this.$message.warning('新密码至少6位')
        return
      }
      this.pwdLoading = true
      try {
        const res = await api.post('/api/user/change-password', {
          old_password: this.pwdForm.oldPassword,
          new_password: this.pwdForm.newPassword
        })
        if (res.data.code === SUCCESS_CODE) {
          this.$message.success('密码修改成功')
          this.profileVisible = false
          this.pwdForm = { oldPassword: '', newPassword: '' }
        }
      } catch {} finally {
        this.pwdLoading = false
      }
    },
    logout() {
      this.$store.dispatch('auth/logout')
      this.$router.push('/login')
    }
  }
}
</script>

<style scoped>
.weekly-container {
  min-height: 100vh;
}
.header {
  background: white;
  padding: 16px 32px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #ebeef5;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}
.header-left h1 {
  font-size: 20px;
  margin: 0;
}
.week-label {
  color: #999;
  font-size: 14px;
}
.week-selector {
  width: 180px;
}
.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}
.username {
  color: #666;
  cursor: pointer;
}
.username:hover {
  color: #409eff;
}
.main-content {
  display: flex;
  padding: 0;
  background: white;
  min-height: calc(100vh - 60px);
}
.fragment-panel {
  flex: 2.5;
  min-width: 0;
  padding: 20px;
  border-right: 1px solid #ebeef5;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 120px);
}
.draft-panel {
  flex: 5;
  min-width: 0;
  padding: 20px;
  border-right: 1px solid #ebeef5;
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: calc(100vh - 120px);
  overflow: hidden;
}
.draft-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  flex-wrap: nowrap;
  overflow: hidden;
  gap: 4px;
}
.draft-topbar-right {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: 4px;
}
.topbar-label {
  font-size: 13px;
  color: #606266;
  white-space: nowrap;
}
.draft-generate-bar {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}
.stream-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
}
.stream-label {
  font-size: 13px;
  color: #606266;
}
.generate-btn {
  background: #4F6EF7;
  border-color: #4F6EF7;
}
.generate-btn:hover {
  background: #3d5bd9;
  border-color: #3d5bd9;
}
.draft-display-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.draft-display-area >>> .el-tabs {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}
.draft-display-area >>> .el-tabs__content {
  flex: 1;
  min-height: 0;
}
.draft-display-area >>> .el-tab-pane {
  height: 100%;
  overflow: auto;
}
.draft-display-area >>> .el-textarea__inner {
  height: 100% !important;
  font-family: 'Menlo', 'Consolas', monospace;
  font-size: 14px;
  line-height: 1.8;
}
.draft-preview-content {
  padding: 16px 16px 16px 32px;
  line-height: 1.8;
  font-size: 14px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  background: #fff;
  overflow: auto;
  box-sizing: border-box;
  min-height: calc(100% - 4px);
}
.draft-preview-content h1, .draft-preview-content h2, .draft-preview-content h3, .draft-preview-content h4 {
  margin: 16px 0 8px;
  color: #303133;
}
.draft-preview-content h3 {
  font-size: 16px;
  border-bottom: 1px solid #ebeef5;
  padding-bottom: 6px;
}
.draft-preview-content p {
  margin: 8px 0;
  color: #606266;
}
.draft-preview-content ul, .draft-preview-content ol {
  padding-left: 24px;
  margin: 8px 0;
}
.draft-preview-content li {
  margin: 4px 0;
  color: #606266;
}
.draft-preview-content strong {
  color: #303133;
}
.draft-preview-content code {
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 13px;
  color: #e6a23c;
}
.draft-preview-content pre {
  background: #f5f7fa;
  padding: 12px 16px;
  border-radius: 4px;
  overflow-x: auto;
}
.draft-preview-content pre code {
  background: none;
  padding: 0;
  color: inherit;
}
.draft-preview-content blockquote {
  border-left: 4px solid #409eff;
  padding: 8px 16px;
  margin: 8px 0;
  background: #f0f9eb;
  color: #67c23a;
}
.draft-preview-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
  color: #909399;
  font-size: 14px;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  background: #fafafa;
}
.panel-title {
  font-size: 16px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  flex-shrink: 0;
}
.fragment-panel-title-right {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.fragment-input {
  flex-shrink: 0;
  margin-bottom: 12px;
}
.fragment-input-inner {
  position: relative;
}
.fragment-date-picker {
  margin-bottom: 6px;
  width: 100%;
}
.fragment-input-inner >>> .el-textarea__inner {
  padding-right: 40px;
  line-height: 1.6;
  resize: none;
  font-family: inherit;
  overflow-x: auto;
  overflow-y: auto;
  white-space: pre;
}
.fragment-input-inner >>> .el-textarea__inner::-webkit-scrollbar {
  height: 6px;
  width: 6px;
}
.fragment-input-inner >>> .el-textarea__inner::-webkit-scrollbar-thumb {
  background: #ddd;
  border-radius: 3px;
}
.fragment-submit-btn {
  position: absolute;
  right: 8px;
  bottom: 8px;
  z-index: 1;
}
.fragment-list {
  flex: 1;
  overflow: auto;
}
.fragment-list::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
.fragment-list::-webkit-scrollbar-thumb {
  background: #ddd;
  border-radius: 3px;
}
.fragment-group {
  margin-bottom: 8px;
}
.fragment-group-title {
  font-size: 12px;
  color: #999;
  padding: 4px 0 4px 4px;
  border-bottom: 1px solid #eee;
  margin-bottom: 4px;
}
.fragment-item {
  display: flex;
  align-items: flex-start;
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 4px;
  background: #f8f9fa;
  transition: background 0.2s;
}
.fragment-item:hover {
  background: #eef0f2;
}
.fragment-body {
  flex: 1;
  min-width: 0;
}
.fragment-date {
  display: block;
  font-size: 11px;
  color: #999;
  margin-bottom: 2px;
}
.carryover-tag {
  margin-right: 4px;
  vertical-align: middle;
}
.fragment-content {
  font-size: 14px;
  white-space: pre;
  line-height: 1.6;
  overflow-x: auto;
}
.fragment-content::-webkit-scrollbar {
  height: 4px;
}
.fragment-content::-webkit-scrollbar-thumb {
  background: #ddd;
  border-radius: 2px;
}
.fragment-index {
  color: #bbb;
  font-size: 12px;
  margin-right: 8px;
  min-width: 20px;
}
.delete-btn {
  padding: 0 4px;
  color: #ccc;
}
.delete-btn:hover {
  color: #f56c6c;
}
.empty-tip {
  text-align: center;
  padding: 40px 0;
  color: #ccc;
}
.empty-tip .el-icon-edit-outline {
  font-size: 32px;
}
.empty-tip p {
  margin-top: 8px;
  font-size: 14px;
}
.fragment-hint {
  flex-shrink: 0;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
  margin-top: 12px;
  color: #bbb;
  text-align: center;
}
.draft-actions-bottom {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}
.draft-bottom-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.finalized-tip {
  color: #67c23a;
  font-weight: 500;
}
.history-panel {
  flex: 2.5;
  min-width: 0;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 60px);
  overflow: hidden;
}
.panel-section {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-bottom: 1px solid #ebeef5;
  padding: 8px 0;
}
.panel-section:last-child { border-bottom: none; }
.section-weekly { flex: 4; }
.section-quarter { flex: 3; }
.section-year { flex: 3; }
.section-header {
  font-size: 14px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
  flex-shrink: 0;
}
.section-header-right {
  display: inline-flex;
  align-items: center;
  gap: 2px;
}
.section-generate {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 6px;
  flex-shrink: 0;
}
.section-scroll {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}
.section-scroll::-webkit-scrollbar { width: 4px; }
.section-scroll::-webkit-scrollbar-thumb { background: #ddd; border-radius: 2px; }
.history-search {
  flex-shrink: 0;
  margin-bottom: 12px;
}
.history-search >>> .el-date-editor {
  width: 100%;
}
.history-list {
  flex: 1;
  overflow-y: auto;
}
.history-list::-webkit-scrollbar {
  width: 6px;
}
.history-list::-webkit-scrollbar-thumb {
  background: #ddd;
  border-radius: 3px;
}
.history-group {
  margin-bottom: 8px;
}
.history-group-title {
  font-size: 12px;
  color: #999;
  padding: 4px 0 4px 4px;
  border-bottom: 1px solid #eee;
  margin-bottom: 4px;
}
.history-item {
  margin-bottom: 6px;
  background: #f8f9fa;
  border-radius: 6px;
  display: flex;
  align-items: flex-start;
  overflow: hidden;
}
.history-item-main {
  flex: 1;
  padding: 10px 12px;
  cursor: pointer;
  transition: background 0.2s;
  min-width: 0;
}
.history-item-main:hover {
  background: #eef0f2;
}
.history-item:hover .history-delete-btn {
  opacity: 1;
}
.history-delete-btn {
  flex-shrink: 0;
  padding: 8px;
  color: #ccc;
  opacity: 0;
  transition: opacity 0.2s;
}
.history-delete-btn:hover {
  color: #f56c6c;
}
.history-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}
.history-week-label {
  font-weight: 600;
  font-size: 13px;
  color: #409eff;
}
.history-item-preview {
  font-size: 13px;
  color: #666;
  line-height: 1.5;
  white-space: pre;
  overflow: hidden;
  text-overflow: ellipsis;
}
.gitlab-commits-list {
  max-height: 300px;
  overflow-y: auto;
  border-top: 1px solid #ebeef5;
  padding-top: 12px;
  margin-top: 8px;
}
.gitlab-commits-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 13px;
  color: #606266;
}
.gitlab-commit-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 8px;
  font-size: 13px;
  border-radius: 4px;
  background: #f5f7fa;
  margin-bottom: 4px;
}
.gitlab-commit-msg {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #303133;
}
.gitlab-commit-date {
  flex-shrink: 0;
  margin-left: 12px;
  color: #909399;
  font-size: 12px;
}
.scroll-loading, .scroll-end {
  text-align: center;
  padding: 12px 0;
  font-size: 13px;
  color: #c0c4cc;
}
.scroll-end {
  color: #dcdfe6;
}
.summary-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
  border-bottom: 1px solid #f0f0f0;
}
.summary-item-main {
  flex: 1;
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  min-width: 0;
}
.summary-item-main:hover {
  background: #f5f7fa;
  margin: 0 -8px;
  padding: 4px 8px;
  border-radius: 4px;
}
.summary-item-label {
  font-size: 13px;
  color: #409eff;
  font-weight: 500;
}
.summary-item-time {
  font-size: 11px;
  color: #c0c4cc;
}
.summary-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 11px;
  color: #909399;
  padding: 2px 0 4px;
  border-bottom: 1px solid #e4e7ed;
}
.summary-delete-btn {
  flex-shrink: 0;
  padding: 0 4px;
  color: #ccc;
  margin-left: 4px;
}
.summary-delete-btn:hover {
  color: #f56c6c;
}
</style>