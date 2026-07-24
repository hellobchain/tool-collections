<template>
  <div style="height: calc(100vh - 110px); display: flex; flex-direction: column;">
    <div class="page-header">
      <h2>🤖 AI 智能问股</h2>
      <p class="page-desc">选择投资策略，与AI助手对话获取专业股票分析建议</p>
    </div>

    <div style="flex:1; display: flex; gap: 8px; min-height: 0;">
      <div style="width: 200px; flex-shrink: 0;">
        <el-card shadow="never" class="compact-card" style="height: 100%;">
          <div slot="header" class="compact-header">策略选择</div>
          <el-radio-group v-model="selectedSkill" style="display:flex;flex-direction:column;">
            <el-radio v-for="s in skills" :key="s.id" :label="s.id" class="skill-item">{{ s.name }}</el-radio>
          </el-radio-group>
          <el-divider style="margin:6px 0;" />
          <div style="font-size:12px;color:#909399;margin-bottom:4px;">历史会话</div>
          <div v-for="s in sessions" :key="s.session_id" class="session-item" @click="loadSession(s.session_id)">
            <div class="session-title">{{ s.title || '新会话' }}</div>
          </div>
          <el-empty v-if="!sessions.length" description="暂无会话" :image-size="60" />
        </el-card>
      </div>

      <div style="flex:1; display: flex; flex-direction: column;">
        <el-card shadow="never" class="compact-card" style="flex:1; display: flex; flex-direction: column;">
          <div slot="header" class="compact-header">
            <span>AI 对话</span>
            <div><el-button size="mini" @click="newSession">新会话</el-button></div>
          </div>
          <div class="chat-messages" ref="chatBox">
            <div v-for="(msg, i) in messages" :key="i" class="chat-msg" :class="msg.role">
              <div class="msg-avatar">{{ msg.role === 'user' ? 'U' : 'AI' }}</div>
              <div class="msg-content" v-html="renderMarkdown(msg.content)"></div>
            </div>
            <div v-if="chatLoading" class="chat-msg assistant">
              <div class="msg-avatar">AI</div>
              <div class="msg-content thinking">思考中...</div>
            </div>
          </div>
          <div class="chat-input">
            <el-input v-model="chatInput" type="textarea" :rows="2" placeholder="输入股票代码或问题..." @keydown.enter.native.prevent="sendMessage" :disabled="chatLoading" />
            <el-button type="primary" @click="sendMessage" :loading="chatLoading" style="height:56px;margin-left:6px;flex-shrink:0;">发送</el-button>
          </div>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script>
import { marked } from 'marked'

export default {
  name: 'AgentChat',
  data() {
    return { chatInput: '', messages: [], sessions: [], skills: [], selectedSkill: 'bull_trend', chatLoading: false, currentSessionId: null }
  },
  mounted() { this.fetchSkills(); this.fetchSessions() },
  methods: {
    async fetchSkills() {
      try {
        await this.$store.dispatch('stock/fetchAgentSkills')
        this.skills = this.$store.state.stock.agentSkills || []
        if (this.skills.length) this.selectedSkill = this.skills[0].id
      } catch {}
    },
    async fetchSessions() {
      try { await this.$store.dispatch('stock/fetchAgentSessions', { limit: 50 }); this.sessions = this.$store.state.stock.agentSessions } catch {}
    },
    async sendMessage() {
      if (!this.chatInput.trim() || this.chatLoading) return
      const msg = this.chatInput.trim(); this.chatInput = ''
      this.messages.push({ role: 'user', content: msg }); this.chatLoading = true; this.scrollToBottom()
      try {
        const res = await this.$store.dispatch('stock/agentChat', { message: msg, session_id: this.currentSessionId, skills: [this.selectedSkill] })
        const data = res?.data?.data
        if (data) { this.currentSessionId = data.session_id; this.messages.push({ role: 'assistant', content: data.content }); this.fetchSessions() }
      } catch { this.messages.push({ role: 'assistant', content: '请求失败，请重试' }) }
      finally { this.chatLoading = false; this.scrollToBottom() }
    },
    async loadSession(sessionId) {
      try { const res = await this.$store.dispatch('stock/getChatSessionMessages', sessionId); const data = res?.data?.data; this.currentSessionId = sessionId; this.messages = data?.messages || [] } catch {}
    },
    newSession() { this.currentSessionId = null; this.messages = [] },
    renderMarkdown(text) { if (!text) return ''; try { return marked(text) } catch { return text } },
    scrollToBottom() { this.$nextTick(() => { const box = this.$refs.chatBox; if (box) box.scrollTop = box.scrollHeight }) }
  }
}
</script>

<style scoped>
.page-header { padding: 12px 0 6px; flex-shrink: 0; }
.page-header h2 { font-size: 18px; margin: 0; color: #303133; }
.page-desc { font-size: 12px; color: #909399; margin: 2px 0 0; }
.compact-card >>> .el-card__header { padding: 6px 10px; }
.compact-card >>> .el-card__body { padding: 8px 10px; }
.compact-header { display: flex; justify-content: space-between; align-items: center; font-size: 13px; font-weight: 500; }
.skill-item { padding: 4px 0; font-size: 12px; }
.session-item { padding: 4px; cursor: pointer; border-bottom: 1px solid #f0f0f0; font-size: 12px; }
.session-item:hover { background: #f5f7fa; }
.session-title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.chat-messages { flex: 1; overflow-y: auto; padding: 8px; }
.chat-msg { display: flex; margin-bottom: 10px; }
.chat-msg.user { flex-direction: row-reverse; }
.msg-avatar { width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 11px; flex-shrink: 0; margin: 0 6px; }
.chat-msg.user .msg-avatar { background: #409eff; color: #fff; }
.chat-msg.assistant .msg-avatar { background: #67c23a; color: #fff; }
.msg-content { max-width: 80%; padding: 6px 10px; border-radius: 6px; font-size: 13px; line-height: 1.5; }
.chat-msg.user .msg-content { background: #ecf5ff; }
.chat-msg.assistant .msg-content { background: #f5f7fa; }
.thinking { color: #909399; font-style: italic; }
.chat-input { padding: 6px; border-top: 1px solid #ebeef5; display: flex; }
</style>
