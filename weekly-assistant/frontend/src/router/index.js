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
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'WeeklyReport',
          component: () => import('@/views/WeeklyReport.vue')
        },
        {
          path: '/document-parse',
          name: 'DocumentParse',
          component: () => import('@/views/DocumentParse.vue')
        }
      ]
    }
  ]
})

router.beforeEach((to, from, next) => {
  if (to.matched.some(r => r.meta.requiresAuth)) {
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