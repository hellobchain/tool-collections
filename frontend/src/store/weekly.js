import api from '@/api'
import { Message } from 'element-ui'
import { FRAGMENT_PAGE_SIZE, HISTORY_PAGE_SIZE, STORAGE_TOKEN, AUTH_SCHEME, ERROR_MSG_DURATION, SUCCESS_CODE, DEFAULT_NARRATIVE } from '@/constants'

export default {
  namespaced: true,
  state: {
    currentWeekStart: '',
    weekStart: null,
    weekEnd: null,
    weekNumber: '',
    fragments: [],
    draftContent: '',
    narrativeType: DEFAULT_NARRATIVE,
    carryover: [],
    carryoverConfirmed: false,
    isGenerating: false,
    isStreaming: false,
    isFinalized: false,
    nextWeekPlan: [],
    history: [],
    historyLoading: false,
    fragmentPage: 1,
    fragmentTotalPages: 1,
    fragmentLoading: false,
    historyPage: 1,
    historyTotalPages: 1
  },
  mutations: {
    SET_CURRENT_WEEK(state, val) { state.currentWeekStart = val },
    SET_WEEK_START(state, date) { state.weekStart = date },
    SET_WEEK_END(state, date) { state.weekEnd = date },
    SET_WEEK_NUMBER(state, val) { state.weekNumber = val },
    SET_FRAGMENTS(state, list) { state.fragments = list },
    APPEND_FRAGMENTS(state, list) { state.fragments = state.fragments.concat(list) },
    ADD_FRAGMENT(state, frag) { state.fragments.push(frag) },
    REMOVE_FRAGMENT(state, id) { state.fragments = state.fragments.filter(f => f.id !== id) },
    SET_FRAGMENT_PAGINATION(state, { page, totalPages }) {
      state.fragmentPage = page
      state.fragmentTotalPages = totalPages
    },
    SET_FRAGMENT_LOADING(state, flag) { state.fragmentLoading = flag },
    SET_DRAFT(state, content) { state.draftContent = content },
    SET_NARRATIVE(state, type) { state.narrativeType = type },
    SET_CARRYOVER(state, list) { state.carryover = list },
    CONFIRM_CARRYOVER(state, val) { state.carryoverConfirmed = val },
    SET_GENERATING(state, flag) { state.isGenerating = flag },
    SET_STREAMING(state, flag) { state.isStreaming = flag },
    SET_FINALIZED(state, flag) { state.isFinalized = flag },
    SET_NEXT_WEEK_PLAN(state, list) { state.nextWeekPlan = list },
    SET_HISTORY(state, list) { state.history = list },
    APPEND_HISTORY(state, list) { state.history = state.history.concat(list) },
    SET_HISTORY_LOADING(state, flag) { state.historyLoading = flag },
    SET_HISTORY_PAGINATION(state, { page, totalPages }) {
      state.historyPage = page
      state.historyTotalPages = totalPages
    }
  },
  actions: {
    async switchWeek({ commit, dispatch }, weekStart) {
      commit('SET_CURRENT_WEEK', weekStart)
      commit('SET_DRAFT', '')
      commit('SET_FRAGMENTS', [])
      commit('SET_FRAGMENT_PAGINATION', { page: 1, totalPages: 1 })
      commit('SET_GENERATING', false)
      commit('SET_STREAMING', false)
      return dispatch('initWeek')
    },
    async initWeek({ commit, state }) {
      try {
        const params = {}
        if (state.currentWeekStart) params.week_start = state.currentWeekStart
        const res = await api.get(`/weekly-assistant/week/status`, { params })
        if (res.data.code === 0) {
          const data = res.data.data
          commit('SET_WEEK_START', data.week_start)
          commit('SET_WEEK_END', data.week_end)
          commit('SET_WEEK_NUMBER', data.week_number)
          commit('SET_FRAGMENTS', data.fragments)
          commit('SET_CARRYOVER', data.carryover || [])
          commit('SET_FINALIZED', data.is_finalized || false)
          commit('SET_NEXT_WEEK_PLAN', data.next_week_plan || [])
          commit('CONFIRM_CARRYOVER', data.is_carryover_confirmed || data.carryover.length === 0)
          return true
        }
      } catch {}
      return false
    },
    loadFragments({ commit, state }, { page, append }) {
      commit('SET_FRAGMENT_LOADING', true)
      return api.get(`/weekly-assistant/fragments`, {
        params: {
          week_start: state.currentWeekStart || state.weekStart,
          page,
          page_size: FRAGMENT_PAGE_SIZE
        }
      }).then(res => {
        if (res.data.code === 0) {
          const data = res.data.data
          if (append) {
            commit('APPEND_FRAGMENTS', data.list)
          } else {
            commit('SET_FRAGMENTS', data.list)
          }
          commit('SET_FRAGMENT_PAGINATION', { page: data.page, totalPages: data.total_pages })
        }
      }).catch(() => {}).finally(() => {
        commit('SET_FRAGMENT_LOADING', false)
      })
    },
    async addFragment({ commit, state }, { content, date }) {
      try {
        const body = { content }
        if (date) {
          body.date = date
        } else if (state.currentWeekStart) {
          body.date = state.currentWeekStart
        }
        const res = await api.post(`/weekly-assistant/fragments`, body)
        if (res.data.code === 0) {
          commit('ADD_FRAGMENT', res.data.data)
          return true
        }
      } catch {}
      return false
    },
    async deleteFragment({ commit }, id) {
      try {
        await api.delete(`/weekly-assistant/fragments/${id}`)
        commit('REMOVE_FRAGMENT', id)
        return true
      } catch {}
      return false
    },
    async confirmCarryover({ commit }, { keptIds, droppedIds }) {
      try {
        await api.post(`/weekly-assistant/week/carryover/confirm`, { kept_ids: keptIds, dropped_ids: droppedIds })
        commit('CONFIRM_CARRYOVER', true)
        return true
      } catch {}
      return false
    },
    async generateDraft({ commit, state }, payload) {
      commit('SET_GENERATING', true)
      commit('SET_DRAFT', '')
      try {
        const body = { narrative_type: state.narrativeType }
        if (payload?.template_id) body.template_id = payload.template_id
        if (state.currentWeekStart) body.week_start = state.currentWeekStart
        const res = await api.post(`/weekly-assistant/week/generate`, body)
        if (res.data.code === 0) {
          commit('SET_DRAFT', res.data.data.content)
        }
      } catch {} finally {
        commit('SET_GENERATING', false)
      }
    },
    async generateDraftStream({ commit, state }, payload) {
      commit('SET_STREAMING', true)
      commit('SET_DRAFT', '')
      try {
		const token = localStorage.getItem(STORAGE_TOKEN)
		const body = { narrative_type: state.narrativeType }
        if (payload?.template_id) body.template_id = payload.template_id
        if (state.currentWeekStart) body.week_start = state.currentWeekStart
        const res = await fetch(`/weekly-assistant/week/generate-stream`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'Authorization': `${AUTH_SCHEME}${token}` },
          body: JSON.stringify(body)
        })
        if (!res.ok) {
          const errText = await res.text().catch(() => '')
          throw new Error(errText || `接口错误: ${res.status}`)
        }
        const reader = res.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        let accumulated = ''
        while (true) {
          const { done, value } = await reader.read()
          if (done) break
          buffer += decoder.decode(value, { stream: true })
          let sepIdx
          while ((sepIdx = buffer.indexOf('\n\n')) !== -1) {
            const event = buffer.slice(0, sepIdx)
            buffer = buffer.slice(sepIdx + 2)
            const dataLine = event.split('\n').find(l => l.startsWith('data: '))
            if (!dataLine) continue
            let data = dataLine.slice(6)
            if (data === '[DONE]') break
            if (data.startsWith('"') && data.endsWith('"')) {
              try { data = JSON.parse(data) } catch {}
            }
            accumulated += data
          }
          commit('SET_DRAFT', accumulated)
        }
      } catch (e) {
        Message.error({ message: e.message || '流式生成失败', duration: ERROR_MSG_DURATION })
      } finally {
        commit('SET_STREAMING', false)
      }
    },
    async finalize({ commit, state }) {
      try {
        const body = { content: state.draftContent, narrative_type: state.narrativeType}
        if (state.currentWeekStart) body.week_start = state.currentWeekStart
        const res = await api.post(`/weekly-assistant/week/finalize`, body)
        if (res.data.code === 0) {
          commit('SET_FINALIZED', true)
          return true
        }
      } catch {}
      return false
    },
    fetchHistory({ commit }, params = {}) {
      commit('SET_HISTORY_LOADING', true)
      const page = params.page || 1
      const query = {}
      if (params.week_start || params.weekStart) query.week_start = params.week_start || params.weekStart
      if (params.week_end || params.weekEnd) query.week_end = params.week_end || params.weekEnd
      query.page = page
      query.page_size = HISTORY_PAGE_SIZE
      return api.get(`/weekly-assistant/week/history`, { params: query }).then(res => {
        if (res.data.code === 0) {
          const data = res.data.data
          if (params.append) {
            commit('APPEND_HISTORY', data.list)
          } else {
            commit('SET_HISTORY', data.list)
          }
          commit('SET_HISTORY_PAGINATION', { page: data.page, totalPages: data.total_pages })
        }
      }).catch(() => {}).finally(() => {
        commit('SET_HISTORY_LOADING', false)
      })
    },
    async deleteReport({ commit, state }, id) {
      try {
        const res = await api.delete(`/weekly-assistant/week/report/${id}`)
        if (res.data.code === 0) {
          commit('SET_HISTORY', state.history.filter(h => h.id !== id))
          return true
        }
      } catch {}
      return false
    }
  },
  getters: {
    fragmentCount: state => state.fragments.length,
    hasCarryover: state => state.carryover.length > 0 && !state.carryoverConfirmed,
    canGenerate: state => state.fragments.length > 0 && !state.isFinalized,
    hasMoreFragments: state => state.fragmentPage < state.fragmentTotalPages,
    hasMoreHistory: state => state.historyPage < state.historyTotalPages
  }
}
