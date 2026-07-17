import api from './index'

export function convertFile(file, toFormats, doOcr = true) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('to_formats', toFormats)
  formData.append('do_ocr', doOcr)

  return api.post('/textparse/v1/convert/fileStream', formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    responseType: 'blob',
    timeout: 180000
  })
}
