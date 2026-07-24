import * as stockApi from '@/api/stock'

export default {
  namespaced: true,
  state: {
    watchlist: [],
    watchlistLoading: false,
    stockBar: [],
    stockBarLoading: false,
    historyList: [],
    historyTotal: 0,
    historyLoading: false,
    currentReport: null,
    tasks: [],
    tasksTotal: 0,
    analysisResult: null,
    analysisLoading: false,
    quote: null,
    quoteLoading: false,
    kline: null,
    klineLoading: false,
    agentSessions: [],
    agentSkills: [],
    chatLoading: false,
    portfolios: [],
    backtestResults: [],
    backtestPerformance: null,
    marketReview: null
  },
  mutations: {
    SET_WATCHLIST(state, list) { state.watchlist = list },
    SET_WATCHLIST_LOADING(state, v) { state.watchlistLoading = v },
    SET_STOCK_BAR(state, data) { state.stockBar = data },
    SET_STOCK_BAR_LOADING(state, v) { state.stockBarLoading = v },
    SET_HISTORY_LIST(state, data) { state.historyList = data },
    SET_HISTORY_TOTAL(state, v) { state.historyTotal = v },
    SET_HISTORY_LOADING(state, v) { state.historyLoading = v },
    SET_CURRENT_REPORT(state, r) { state.currentReport = r },
    SET_TASKS(state, data) { state.tasks = data },
    SET_TASKS_TOTAL(state, v) { state.tasksTotal = v },
    SET_ANALYSIS_RESULT(state, r) { state.analysisResult = r },
    SET_ANALYSIS_LOADING(state, v) { state.analysisLoading = v },
    SET_QUOTE(state, q) { state.quote = q },
    SET_QUOTE_LOADING(state, v) { state.quoteLoading = v },
    SET_KLINE(state, k) { state.kline = k },
    SET_KLINE_LOADING(state, v) { state.klineLoading = v },
    SET_AGENT_SESSIONS(state, s) { state.agentSessions = s },
    SET_AGENT_SKILLS(state, s) { state.agentSkills = s },
    SET_CHAT_LOADING(state, v) { state.chatLoading = v },
    SET_PORTFOLIOS(state, p) { state.portfolios = p },
    SET_BACKTEST_RESULTS(state, r) { state.backtestResults = r },
    SET_BACKTEST_PERFORMANCE(state, p) { state.backtestPerformance = p },
    SET_MARKET_REVIEW(state, r) { state.marketReview = r }
  },
  actions: {
    async fetchWatchlist({ commit }) {
      commit('SET_WATCHLIST_LOADING', true)
      try {
        const res = await stockApi.getWatchlist()
        commit('SET_WATCHLIST', res.data.data?.stock_codes || [])
      } finally { commit('SET_WATCHLIST_LOADING', false) }
    },
    async addWatchlist({ dispatch }, code) {
      await stockApi.addToWatchlist(code)
      await dispatch('fetchWatchlist')
    },
    async removeWatchlist({ dispatch }, code) {
      await stockApi.removeFromWatchlist(code)
      await dispatch('fetchWatchlist')
    },
    async fetchStockBar({ commit }, params) {
      commit('SET_STOCK_BAR_LOADING', true)
      try {
        const res = await stockApi.getStockBar(params)
        commit('SET_STOCK_BAR', res.data.data?.items || [])
      } finally { commit('SET_STOCK_BAR_LOADING', false) }
    },
    async fetchHistoryList({ commit }, params) {
      commit('SET_HISTORY_LOADING', true)
      try {
        const res = await stockApi.getStockHistoryList(params)
        const d = res.data.data || {}
        commit('SET_HISTORY_LIST', d.list || [])
        commit('SET_HISTORY_TOTAL', d.total || 0)
      } finally { commit('SET_HISTORY_LOADING', false) }
    },
    async fetchHistoryDetail({ commit }, id) {
      const res = await stockApi.getStockHistoryDetail(id)
      commit('SET_CURRENT_REPORT', res.data.data)
    },
    async deleteHistory({ commit, dispatch }, params) {
      await stockApi.deleteStockHistory(params)
      await dispatch('fetchHistoryList', { page: 1, limit: 20 })
    },
    async triggerAnalysis({ commit }, params) {
      commit('SET_ANALYSIS_LOADING', true)
      try {
        const res = await stockApi.triggerAnalysis(params)
        commit('SET_ANALYSIS_RESULT', res.data.data)
        return res.data.data
      } finally { commit('SET_ANALYSIS_LOADING', false) }
    },
    async fetchQuote({ commit }, code) {
      commit('SET_QUOTE_LOADING', true)
      try {
        const res = await stockApi.getStockQuote(code)
        const data = res.data.data
        commit('SET_QUOTE', data)
        return data
      } finally { commit('SET_QUOTE_LOADING', false) }
    },
    async fetchKline({ commit }, { code, period, days }) {
      commit('SET_KLINE_LOADING', true)
      try {
        const res = await stockApi.getStockHistory(code, { period, days })
        const data = res.data.data
        commit('SET_KLINE', data)
        return data
      } finally { commit('SET_KLINE_LOADING', false) }
    },
    async agentChat({ commit }, params) {
      commit('SET_CHAT_LOADING', true)
      try {
        return await stockApi.agentChat(params)
      } finally { commit('SET_CHAT_LOADING', false) }
    },
    async getChatSessionMessages({ commit }, sessionId) {
      return await stockApi.getChatSessionMessages(sessionId)
    },
    async deleteChatSession({ commit }, sessionId) {
      return await stockApi.deleteChatSession(sessionId)
    },
    async importStockCodesText({ commit }, text) {
      const res = await stockApi.importStockCodesText(text)
      return res
    },
    async fetchAgentSessions({ commit }, params) {
      const res = await stockApi.listChatSessions(params)
      commit('SET_AGENT_SESSIONS', res.data.data?.sessions || [])
    },
    async fetchAgentSkills({ commit }) {
      const res = await stockApi.listAgentSkills()
      commit('SET_AGENT_SKILLS', res.data.data?.skills || [])
    },
    async fetchPortfolios({ commit }) {
      const res = await stockApi.listPortfolioAccounts()
      commit('SET_PORTFOLIOS', res.data.data?.accounts || [])
    },
    async fetchBacktestResults({ commit }, params) {
      const res = await stockApi.getBacktestResults(params)
      const d = res.data.data || {}
      commit('SET_BACKTEST_RESULTS', d.list || [])
    },
    async fetchBacktestPerformance({ commit }) {
      const res = await stockApi.getBacktestPerformance()
      commit('SET_BACKTEST_PERFORMANCE', res.data.data)
    },
    async triggerMarketReview({ commit }, params) {
      const res = await stockApi.triggerMarketReview(params)
      commit('SET_MARKET_REVIEW', res.data.data)
      return res.data.data
    }
  }
}
