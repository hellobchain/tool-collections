import api from './index'
import {API_PREX} from '@/constants'

export function uploadContract(file, onProgress) {
  const formData = new FormData()
  formData.append('file', file)
  return api.post(API_PREX+'/contract/v1/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: onProgress,
    timeout: 300000
  })
}

export function deleteContract(fileId) {
  return api.delete(API_PREX+`/contract/v1/files/${fileId}`)
}

export function startReview(params) {
  return api.post(API_PREX+`/contract/v1/review`, params, { timeout: 300000 })
}

export function getReviewProgress(taskId) {
  return api.get(API_PREX+`/contract/v1/review/${taskId}/progress`)
}

export function getReviewReport(reportId) {
  return api.get(API_PREX+`/contract/v1/report/${reportId}`)
}

export function updateReviewItem(reportId, itemId, payload) {
  return api.put(API_PREX+`/contract/v1/report/${reportId}/items/${itemId}`, payload)
}

export function getContractText(fileId) {
  return api.get(API_PREX+`/contract/v1/files/${fileId}/text`)
}

export function getHistory(params) {
  return api.get(API_PREX+'/contract/v1/history', { params })
}

export function deleteHistory(reportId) {
  return api.delete(API_PREX+`/contract/v1/history/${reportId}`)
}

export function startDraftGen(params) {
  return api.post(API_PREX+'/contract/v1/draft/generate', params, { timeout: 300000 })
}

export function getDraftProgress(taskId) {
  return api.get(API_PREX+`/contract/v1/draft/${taskId}/progress`)
}

export function getDraftResult(taskId) {
  return api.get(API_PREX+`/contract/v1/draft/${taskId}/result`)
}

export function downloadDraft(taskId) {
  return api.get(API_PREX+`/contract/v1/draft/${taskId}/download`, { responseType: 'blob' })
}

export function getDraftHistory(params) {
  return api.get(API_PREX+'/contract/v1/draft/history', { params })
}

export function getDraftDetail(draftId) {
  return api.get(API_PREX+`/contract/v1/draft/history/${draftId}`)
}

export function startExtract(params) {
  return api.post(API_PREX+'/contract/v1/extract/start', params, { timeout: 300000 })
}
export function getExtractProgress(taskId) {
  return api.get(API_PREX+`/contract/v1/extract/${taskId}/progress`)
}
export function getExtractResult(taskId) {
  return api.get(API_PREX+`/contract/v1/extract/${taskId}/result`)
}
export function updateExtractCell(resultId, field, value) {
  return api.put(API_PREX+`/contract/v1/extract/result/${resultId}/cell`, { field, value })
}
export function getExtractHistory(params) {
  return api.get(API_PREX+'/contract/v1/extract/history', { params })
}
export function deleteExtractTask(taskId) {
  return api.delete(API_PREX+`/contract/v1/extract/${taskId}`)
}
export function exportExtractResult(taskId) {
  return api.get(API_PREX+`/contract/v1/extract/${taskId}/export`, { responseType: 'blob' })
}

export function deleteDraftHistory(draftId) {
  return api.delete(API_PREX+`/contract/v1/draft/history/${draftId}`)
}

export function exportReport(reportId, format) {
  return api.get(API_PREX+`/contract/v1/report/${reportId}/export`, {
    params: { format },
    responseType: 'blob'
  })
}
