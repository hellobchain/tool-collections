import api from './index'

export function langConvert(jsonData, language) {
  return api.post('/weekly-assistant/json-tool/v1/lang-convert', {
    json_data: jsonData,
    language: language
  })
}
