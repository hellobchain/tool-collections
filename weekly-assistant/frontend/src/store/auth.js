import api from '@/api'
import { STORAGE_TOKEN, STORAGE_USER, SUCCESS_CODE } from '@/constants'

export default {
  namespaced: true,
  state: {
    token: localStorage.getItem(STORAGE_TOKEN) || null,
    user: JSON.parse(localStorage.getItem(STORAGE_USER) || 'null')
  },
  mutations: {
    SET_AUTH(state, { token, user }) {
      state.token = token
      state.user = user
      localStorage.setItem(STORAGE_TOKEN, token)
      localStorage.setItem(STORAGE_USER, JSON.stringify(user))
    },
    CLEAR_AUTH(state) {
      state.token = null
      state.user = null
      localStorage.removeItem(STORAGE_TOKEN)
      localStorage.removeItem(STORAGE_USER)
    }
  },
  actions: {
    async login({ commit }, { username, password }) {
      try {
        const res = await api.post(`/weekly-assistant/auth/login`, { username, password })
        if (res.data.code === SUCCESS_CODE) {
          commit('SET_AUTH', res.data.data)
          return true
        }
        return false
      } catch {
        return false
      }
    },
    logout({ commit }) {
      commit('CLEAR_AUTH')
    }
  },
  getters: {
    isAuthenticated: state => !!state.token,
    currentUser: state => state.user
  }
}