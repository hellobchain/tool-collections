export default {
  namespaced: true,
  state: {
    carryoverDialogVisible: false,
    toastMessage: null
  },
  mutations: {
    SHOW_CARRYOVER_DIALOG(state) { state.carryoverDialogVisible = true },
    HIDE_CARRYOVER_DIALOG(state) { state.carryoverDialogVisible = false },
    SET_TOAST(state, msg) { state.toastMessage = msg }
  }
}