import api from './index'

export function uploadContract(file, onProgress) {
  const formData = new FormData()
  formData.append('file', file)
  return api.post('/contract/v1/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: onProgress,
    timeout: 300000
  })
}

export function deleteContract(fileId) {
  return api.delete(`/contract/v1/files/${fileId}`)
}

export function startReview(params) {
  return api.post('/contract/v1/review', params, { timeout: 300000 })
}

export function getReviewProgress(taskId) {
  return api.get(`/contract/v1/review/${taskId}/progress`)
}

export function getReviewReport(reportId) {
  return api.get(`/contract/v1/report/${reportId}`)
}

export function updateReviewItem(reportId, itemId, payload) {
  return api.put(`/contract/v1/report/${reportId}/items/${itemId}`, payload)
}

export function getContractText(fileId) {
  return api.get(`/contract/v1/files/${fileId}/text`)
}

export function getHistory(params) {
  return api.get('/contract/v1/history', { params })
}

export function deleteHistory(reportId) {
  return api.delete(`/contract/v1/history/${reportId}`)
}

export function exportReport(reportId, format) {
  return api.get(`/contract/v1/report/${reportId}/export`, {
    params: { format },
    responseType: 'blob'
  })
}
