import axios from 'axios'
import { Message } from 'element-ui'
import { AXIOS_TIMEOUT, STORAGE_TOKEN, STORAGE_USER, AUTH_SCHEME, ERROR_MSG_DURATION, SUCCESS_CODE,NOT_FOUND_CODE,UNAUTHORIZED_CODE } from '@/constants'

const instance = axios.create({
  baseURL: process.env.VUE_APP_API_BASE_URL,
  timeout: AXIOS_TIMEOUT
})

instance.interceptors.request.use(config => {
  const token = localStorage.getItem(STORAGE_TOKEN)
  if (token) {
    config.headers.Authorization = `${AUTH_SCHEME}${token}`
  }
  return config
})

// 响应拦截器
instance.interceptors.response.use(
  (response) => {
    if (response.config.responseType === 'blob' || response.config.responseType === 'arraybuffer') { 
      return response;
    }
    const { code, msg, data } = response.data
    // 业务成功
    if (code === SUCCESS_CODE) {
      return response
    }
    
    // 业务失败 - 统一处理
    if (code === UNAUTHORIZED_CODE) {
      localStorage.removeItem(STORAGE_TOKEN)
      localStorage.removeItem(STORAGE_USER)
      window.location.href = '/login'
      Message.error({ message: msg || '登录已过期，请重新登录' })
      return Promise.reject(new Error(msg))
    }
    
    if (code === NOT_FOUND_CODE) {
      Message.error({ message: msg || `接口不存在: ${response.config.url}` })
      return Promise.reject(new Error(msg))
    }
    
    // 其他业务错误
    Message.error({ message: msg || '请求失败' })
    return Promise.reject(new Error(msg))
  },
  
  // 网络错误/HTTP非200才进入这里
  (error) => {
    Message.error({ message: '网络异常，请稍后重试' })
    return Promise.reject(error)
  }
)

export default instance