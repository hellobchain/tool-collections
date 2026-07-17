import api from './index'

export default {
    // 获取所有模板，支持分页
    getTemplates(params = {}) {
        return api.get(`/weekly-assistant/prompts`, { params })
    },
    // 获取单个模板
    getTemplate(id) {
        return api.get(`/weekly-assistant/prompts/${id}`)
    },
    // 创建模板
    createTemplate(data) {
        return api.post(`/weekly-assistant/prompts`, data)
    },
    // 更新模板
    updateTemplate(id, data) {
        return api.put(`/weekly-assistant/prompts/${id}`, data)
    },
    // 删除模板
    deleteTemplate(id) {
        return api.delete(`/weekly-assistant/prompts/${id}`)
    }
}