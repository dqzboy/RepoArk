import { createRouter, createWebHistory } from 'vue-router'
import Login from '../views/Login.vue'
import Layout from '../layout/Layout.vue'

const Dashboard = () => import('../views/Dashboard.vue')
const Config = () => import('../views/Config.vue')
const Backup = () => import('../views/Backup.vue')
const Jobs = () => import('../views/Jobs.vue')
const Users = () => import('../views/Users.vue')
const Schedule = () => import('../views/Schedule.vue')

const routes = [
  { path: '/login', name: 'login', component: Login },
  {
    path: '/',
    component: Layout,
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'dashboard', component: Dashboard, meta: { title: 'nav.dashboard' } },
      { path: 'config', name: 'config', component: Config, meta: { title: 'nav.config' } },
      { path: 'backup', name: 'backup', component: Backup, meta: { title: 'nav.backup' } },
      { path: 'jobs', name: 'jobs', component: Jobs, meta: { title: 'nav.jobs' } },
      { path: 'users', name: 'users', component: Users, meta: { title: 'nav.users' } },
      { path: 'schedule', name: 'schedule', component: Schedule, meta: { title: 'nav.schedule' } }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    return '/login'
  }
  if (to.path === '/login' && token) {
    return '/'
  }
})

export default router
