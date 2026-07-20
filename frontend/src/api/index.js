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
  // 在成功回调中判断业务 code
  (response) => {
    // 二进制流直接返回
    if (response.config.responseType === 'blob' || response.config.responseType === 'arraybuffer') { 
      try {
        // 尝试将二进制数据转为文本
        const text = response.data; // 如果是 blob
        const json = JSON.parse(text);
        // 如果解析成功且包含 code，说明是业务错误
        if (json.code && json.code !== SUCCESS_CODE) {
          // 构造错误对象，保持与正常错误处理一致
          const error = new Error(json.msg || '请求失败');
          error.code = json.code;
          error.data = json;
          Message.error({ message: json.msg || '请求失败' });
          return Promise.reject(error);
        }
        // 如果解析成功且 code 正确，说明是正常的二进制响应，但这种情况极少
        // 绝大多数情况下，业务错误才会用 JSON，正常文件流不会解析为 JSON
        return response;
      } catch (e) {
        // 解析失败，说明是真正的二进制文件流，正常返回
        return response;
      }
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