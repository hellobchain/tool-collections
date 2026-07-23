import api from './index'

export function jsonCompare(jsonA, jsonB) {
  return api.post('/weekly-assistant/json-compare/v1/compare', { json_a: jsonA, json_b: jsonB })
}

export function convertFile(file, toFormats, doOcr = true) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('to_formats', toFormats)
  formData.append('do_ocr', doOcr)

  return api.post(`/textparse/v1/convert/fileStream`, formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    responseType: 'blob',
    timeout: 180000
  })
}

export function mdFile2DocxFile(file) {
  const formData = new FormData()
  formData.append('file', file)

  return api.post(`/md2docx/v1/convert/file`, formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    responseType: 'blob',
    timeout: 180000
  })
}

export function detectDocType(file) {
  const formData = new FormData()
  formData.append('file', file)
  return api.post('/weekly-assistant/doc-type/v1/detect', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}