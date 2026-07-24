import api from './index'

export function triggerAnalysis(params) {
  return api.post('/weekly-assistant/stock/v1/analysis/analyze', params, { timeout: 300000 })
}

export function getAnalysisStatus(taskId) {
  return api.get(`/weekly-assistant/stock/v1/analysis/status/${taskId}`)
}

export function listAnalysisTasks(params) {
  return api.get('/weekly-assistant/stock/v1/analysis/tasks', { params })
}

export function triggerMarketReview(params) {
  return api.post('/weekly-assistant/stock/v1/analysis/market-review', params || {}, { timeout: 120000 })
}

export function getAnalysisTaskFlow(taskId) {
  return api.get(`/weekly-assistant/stock/v1/analysis/tasks/${taskId}/flow`)
}

export function getStockQuote(code) {
  return api.get(`/weekly-assistant/stock/v1/stocks/quote/${code}`)
}

export function getStockHistory(code, params) {
  return api.get(`/weekly-assistant/stock/v1/stocks/history/${code}`, { params })
}

export function getWatchlist() {
  return api.get('/weekly-assistant/stock/v1/stocks/watchlist')
}

export function addToWatchlist(stockCode) {
  return api.post('/weekly-assistant/stock/v1/stocks/watchlist/add', { stock_code: stockCode })
}

export function removeFromWatchlist(stockCode) {
  return api.post('/weekly-assistant/stock/v1/stocks/watchlist/remove', { stock_code: stockCode })
}

export function importStockCodes(formData) {
  return api.post('/weekly-assistant/stock/v1/stocks/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 60000
  })
}

export function importStockCodesText(text) {
  return api.post('/weekly-assistant/stock/v1/stocks/import', { text })
}

export function getStockHistoryList(params) {
  return api.get('/weekly-assistant/stock/v1/history', { params })
}

export function getStockBar(params) {
  return api.get('/weekly-assistant/stock/v1/history/stocks', { params })
}

export function getStockHistoryDetail(id) {
  return api.get(`/weekly-assistant/stock/v1/history/${id}`)
}

export function deleteStockHistory(recordIds) {
  return api.delete('/weekly-assistant/stock/v1/history', { data: { record_ids: recordIds } })
}

export function deleteStockHistoryByCode(code) {
  return api.delete(`/weekly-assistant/stock/v1/history/by-code/${code}`)
}

export function getStockHistoryMarkdown(id) {
  return api.get(`/weekly-assistant/stock/v1/history/${id}/markdown`)
}

export function agentChat(params) {
  return api.post('/weekly-assistant/stock/v1/agent/chat', params, { timeout: 120000 })
}

export function listAgentSkills() {
  return api.get('/weekly-assistant/stock/v1/agent/skills')
}

export function listChatSessions(params) {
  return api.get('/weekly-assistant/stock/v1/agent/chat/sessions', { params })
}

export function getChatSessionMessages(sessionId) {
  return api.get(`/weekly-assistant/stock/v1/agent/chat/sessions/${sessionId}`)
}

export function deleteChatSession(sessionId) {
  return api.delete(`/weekly-assistant/stock/v1/agent/chat/sessions/${sessionId}`)
}

export function createPortfolioAccount(params) {
  return api.post('/weekly-assistant/stock/v1/portfolio/accounts', params)
}

export function listPortfolioAccounts() {
  return api.get('/weekly-assistant/stock/v1/portfolio/accounts')
}

export function recordPortfolioTrade(params) {
  return api.post('/weekly-assistant/stock/v1/portfolio/trades', params)
}

export function getPortfolioSnapshot() {
  return api.get('/weekly-assistant/stock/v1/portfolio/snapshot')
}

export function runBacktest(params) {
  return api.post('/weekly-assistant/stock/v1/backtest/run', params || {}, { timeout: 120000 })
}

export function getBacktestResults(params) {
  return api.get('/weekly-assistant/stock/v1/backtest/results', { params })
}

export function getBacktestPerformance(params) {
  return api.get('/weekly-assistant/stock/v1/backtest/performance', { params })
}

export function getStockSystemConfig() {
  return api.get('/weekly-assistant/stock/v1/system/config')
}

export function updateStockSystemConfig(params) {
  return api.put('/weekly-assistant/stock/v1/system/config', params)
}
