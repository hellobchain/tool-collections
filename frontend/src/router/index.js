import Vue from 'vue'
import Router from 'vue-router'
import store from '@/store'

Vue.use(Router)

const router = new Router({
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue')
    },
    {
      path: '/',
      name: 'WeeklyReport',
      component: () => import('@/views/WeeklyReport.vue'),
      meta: { requiresAuth: true }
    }
  ]
})

router.beforeEach((to, from, next) => {
  if (to.meta.requiresAuth) {
    if (store.getters['auth/isAuthenticated']) {
      next()
    } else {
      next('/login')
    }
  } else {
    next()
  }
})

export default router