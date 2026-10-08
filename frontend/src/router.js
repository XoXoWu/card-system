import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/login', component: () => import('./views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('./views/Layout.vue'),
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', component: () => import('./views/Dashboard.vue'), meta: { title: '数据看板' } },
      { path: 'generate', component: () => import('./views/Generate.vue'), meta: { title: '卡密生成' } },
      { path: 'cards', component: () => import('./views/Cards.vue'), meta: { title: '卡密列表' } },
      { path: 'plans', component: () => import('./views/Plans.vue'), meta: { title: '套餐管理' } },
      { path: 'oplogs', component: () => import('./views/OpLogs.vue'), meta: { title: '操作日志' } },
      { path: 'settings', component: () => import('./views/Settings.vue'), meta: { title: '系统设置' } }
    ]
  }
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach(to => {
  if (!to.meta.public && !localStorage.getItem('token')) return '/login'
})

export default router
