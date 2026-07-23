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
        },
        {
          path: '/json-tool',
          name: 'JsonTool',
          component: () => import('@/views/JsonTool.vue')
        },
        {
          path: '/contract-review',
          name: 'ContractReview',
          component: () => import('@/views/ContractReview.vue')
        },
        {
          path: '/contract-history',
          name: 'ContractHistory',
          component: () => import('@/views/ContractHistory.vue')
        },
        {
          path: '/contract-draft',
          name: 'ContractDraft',
          component: () => import('@/views/ContractDraft.vue')
        },
        {
          path: '/contract-draft-history',
          name: 'ContractDraftHistory',
          component: () => import('@/views/ContractDraftHistory.vue')
        },
        {
          path: '/contract-extract',
          name: 'ContractExtract',
          component: () => import('@/views/ContractExtract.vue')
        },
        {
          path: '/contract-extract-history',
          name: 'ContractExtractHistory',
          component: () => import('@/views/ContractExtractHistory.vue')
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