import Vue from 'vue'
import Vuex from 'vuex'
import auth from './auth'
import weekly from './weekly'
import ui from './ui'
import contract from './contract'

Vue.use(Vuex)

export default new Vuex.Store({
  modules: {
    auth,
    weekly,
    ui,
    contract
  }
})