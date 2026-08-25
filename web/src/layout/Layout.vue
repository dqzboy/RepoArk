<template>
  <div class="archive-shell">
    <a class="skip-link" href="#main-content">{{ t('layout.skip') }}</a>
    <div v-if="isMobile && sidebarOpen" class="archive-backdrop" @click="sidebarOpen = false" />

    <aside class="archive-sidebar" :class="{ 'is-open': isMobile && sidebarOpen }" aria-label="Primary">
      <router-link class="archive-brand" to="/dashboard" @click="sidebarOpen = false">
        <BrandMark />
        <span><strong>{{ t('app.brand') }}</strong><small>{{ t('app.sub') }}</small></span>
      </router-link>

      <div class="archive-nav-label">{{ t('layout.navigation') }}</div>
      <nav class="archive-nav">
        <router-link v-for="(item, index) in navItems" :key="item.path" :to="item.path" class="archive-nav__item" @click="sidebarOpen = false">
          <span class="archive-nav__index">0{{ index + 1 }}</span>
          <el-icon><component :is="item.icon" /></el-icon>
          <span class="archive-nav__text">{{ t(item.label) }}</span>
          <span class="archive-nav__rail" />
        </router-link>
      </nav>

      <div class="archive-side-status">
        <div class="archive-side-status__head"><span>{{ t('layout.archiveNode') }}</span><span class="status-light" /></div>
        <strong>{{ t('app.online') }}</strong>
        <a :href="repoUrl" target="_blank" rel="noopener" class="archive-repo-link">
          <svg viewBox="0 0 16 16" fill="currentColor" aria-hidden="true"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0 0 16 8c0-4.42-3.58-8-8-8Z" /></svg>
          <span>{{ repoName }}</span><el-icon><TopRight /></el-icon>
        </a>
      </div>
    </aside>

    <section class="archive-workspace">
      <header class="archive-header">
        <div class="archive-header__identity">
          <button v-if="isMobile" class="icon-control" type="button" :aria-label="t('responsive.menu')" @click="sidebarOpen = !sidebarOpen"><el-icon><Menu /></el-icon></button>
          <div><div class="archive-kicker">{{ t('layout.controlRoom') }} / {{ current }}</div><h1>{{ pageTitle }}</h1></div>
        </div>

        <div class="archive-header__actions">
          <router-link class="header-run" to="/backup"><el-icon><Upload /></el-icon><span>{{ t('nav.backup') }}</span></router-link>
          <el-dropdown trigger="click" @command="onLang">
            <button class="text-control" type="button" :aria-label="t('lang.label')">{{ currentLangLabel }}<el-icon><ArrowDown /></el-icon></button>
            <template #dropdown>
              <el-dropdown-menu class="ops-lang-menu">
                <el-dropdown-item v-for="item in availableLocales" :key="item.code" :command="item.code" :class="{ 'is-active': locale === item.code }">
                  <el-icon v-if="locale === item.code"><Select /></el-icon><span v-else class="dropdown-placeholder" />{{ item.label }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <button class="icon-control" type="button" :aria-label="t('layout.toggleTheme')" @click="onToggle"><el-icon><Sunny v-if="theme === 'dark'" /><Moon v-else /></el-icon></button>
          <el-dropdown @command="onCommand">
            <button class="user-control" type="button"><span class="archive-avatar">{{ initial }}</span><span class="archive-user-name">{{ username }}</span><el-icon><ArrowDown /></el-icon></button>
            <template #dropdown><el-dropdown-menu><el-dropdown-item command="logout">{{ t('common.logout') }}</el-dropdown-item></el-dropdown-menu></template>
          </el-dropdown>
        </div>
      </header>

      <main id="main-content" class="archive-main" tabindex="-1">
        <router-view v-slot="{ Component }"><transition name="ops-route" mode="out-in"><component :is="Component" /></transition></router-view>
      </main>
    </section>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowDown, Clock, DataLine, List, Menu, Moon, Select, Setting, Sunny, TopRight, Upload, User } from '@element-plus/icons-vue'
import BrandMark from '../components/BrandMark.vue'
import { getStoredTheme, toggleTheme } from '../theme'
import { GITHUB_REPO_NAME, GITHUB_REPO_URL } from '../github'
import { t, useI18n } from '../i18n'

const route = useRoute()
const router = useRouter()
const { locale, setLocale, availableLocales } = useI18n()
const navItems = [
  { path: '/dashboard', label: 'nav.dashboard', icon: DataLine },
  { path: '/config', label: 'nav.config', icon: Setting },
  { path: '/backup', label: 'nav.backup', icon: Upload },
  { path: '/jobs', label: 'nav.jobs', icon: List },
  { path: '/schedule', label: 'nav.schedule', icon: Clock },
  { path: '/users', label: 'nav.users', icon: User }
]
const current = computed(() => route.path.replace('/', '') || 'dashboard')
const pageTitle = computed(() => t(route.meta.title || 'app.name'))
const username = localStorage.getItem('username') || 'admin'
const initial = username.charAt(0).toUpperCase()
const theme = ref(getStoredTheme() || 'dark')
const repoUrl = GITHUB_REPO_URL
const repoName = GITHUB_REPO_NAME
const isMobile = ref(false)
const sidebarOpen = ref(false)
let mq
const currentLangLabel = computed(() => ({ 'zh-CN': '简体', 'zh-TW': '繁體', en: 'EN' }[locale.value] || '简体'))

function applyMQ(event) { isMobile.value = event.matches; if (!event.matches) sidebarOpen.value = false }
function onLang(code) { setLocale(code) }
function onToggle() { theme.value = toggleTheme() }
function onCommand(command) {
  if (command !== 'logout') return
  localStorage.removeItem('token'); localStorage.removeItem('username'); router.push('/login')
}
onMounted(() => { mq = window.matchMedia('(max-width: 920px)'); applyMQ(mq); mq.addEventListener('change', applyMQ) })
onBeforeUnmount(() => mq?.removeEventListener('change', applyMQ))
</script>

<style scoped>
.archive-shell { min-height: 100vh; display: grid; grid-template-columns: 272px minmax(0, 1fr); }
.skip-link { position: fixed; left: 16px; top: -80px; z-index: 100; padding: 10px 14px; background: var(--accent); color: var(--accent-ink); border-radius: 6px; }
.skip-link:focus { top: 16px; }
.archive-sidebar { position: sticky; top: 0; height: 100vh; display: flex; flex-direction: column; box-sizing: border-box; padding: 22px 16px 16px; background: var(--sidebar); color: var(--sidebar-text); border-right: 1px solid var(--sidebar-border); z-index: 30; }
.archive-brand { display: flex; align-items: center; gap: 14px; min-height: 56px; color: inherit; text-decoration: none; }
.archive-brand strong { display: block; font: 700 14px/1.1 var(--font-display); letter-spacing: .06em; }
.archive-brand small { display: block; margin-top: 5px; color: var(--sidebar-muted); font: 500 10px/1 var(--font-display); letter-spacing: .18em; text-transform: uppercase; }
.archive-nav-label { margin: 38px 10px 10px; color: var(--sidebar-muted); font: 600 10px/1 var(--font-display); letter-spacing: .2em; text-transform: uppercase; }
.archive-nav { display: grid; gap: 4px; }
.archive-nav__item { position: relative; min-height: 48px; display: grid; grid-template-columns: 26px 24px 1fr; align-items: center; gap: 9px; padding: 0 12px; color: var(--sidebar-muted); text-decoration: none; border-radius: 8px; overflow: hidden; transition: color .2s ease, background-color .2s ease; }
.archive-nav__item:hover { color: var(--sidebar-text); background: var(--sidebar-hover); }
.archive-nav__item.router-link-active { color: var(--sidebar-text); background: var(--sidebar-active); }
.archive-nav__item:focus-visible { outline: 3px solid var(--focus); outline-offset: 2px; }
.archive-nav__index { font: 600 9px/1 var(--font-display); letter-spacing: .08em; color: var(--sidebar-muted); }
.archive-nav__text { font: 600 13px/1 var(--font-display); }
.archive-nav__rail { position: absolute; right: 0; top: 10px; bottom: 10px; width: 3px; background: var(--accent); opacity: 0; }
.router-link-active .archive-nav__rail { opacity: 1; }
.archive-side-status { margin-top: auto; padding: 16px; color: var(--sidebar-muted); background: var(--sidebar-panel); border: 1px solid var(--sidebar-border); border-radius: 10px 10px 10px 3px; }
.archive-side-status__head { display: flex; justify-content: space-between; font: 600 9px/1 var(--font-display); letter-spacing: .18em; text-transform: uppercase; }
.archive-side-status strong { display: block; margin: 9px 0 14px; color: var(--sidebar-text); font: 600 12px/1 var(--font-display); }
.status-light { width: 8px; height: 8px; border-radius: 50%; background: var(--accent); box-shadow: 0 0 0 4px color-mix(in srgb, var(--accent) 16%, transparent); }
.archive-repo-link { min-height: 44px; display: flex; align-items: center; gap: 9px; color: inherit; text-decoration: none; font-size: 12px; border-top: 1px solid var(--sidebar-border); padding-top: 10px; }
.archive-repo-link:hover { color: var(--accent); }
.archive-repo-link svg { width: 17px; height: 17px; }
.archive-repo-link span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.archive-workspace { min-width: 0; }
.archive-header { min-height: 82px; display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 0 clamp(18px, 3vw, 40px); background: color-mix(in srgb, var(--bg) 92%, transparent); border-bottom: 1px solid var(--border); backdrop-filter: blur(14px); }
.archive-header__identity, .archive-header__actions { display: flex; align-items: center; gap: 10px; min-width: 0; }
.archive-header__identity { flex: 1 1 auto; }
.archive-header__actions { flex: 0 0 auto; }
.archive-kicker { color: var(--text-faint); font: 600 9px/1 var(--font-display); letter-spacing: .18em; text-transform: uppercase; }
.archive-header h1 { margin: 6px 0 0; color: var(--text); font: 650 clamp(18px, 2vw, 24px)/1.1 var(--font-body); }
.header-run, .text-control, .icon-control, .user-control { min-height: 44px; flex: 0 0 auto; display: inline-flex; align-items: center; justify-content: center; gap: 8px; box-sizing: border-box; border: 1px solid var(--border); border-radius: 8px; cursor: pointer; transition: color .2s ease, background-color .2s ease, border-color .2s ease; }
.header-run { padding: 0 15px; color: var(--accent-ink); background: var(--accent); border-color: var(--accent); text-decoration: none; font: 650 12px/1 var(--font-display); }
.header-run:hover { background: var(--accent-strong); }
.text-control, .icon-control, .user-control { background: var(--surface); color: var(--text-muted); }
.text-control { padding: 0 12px; font: 600 12px/1 var(--font-display); }
.icon-control { width: 44px; padding: 0; }
.user-control { padding: 0 9px 0 5px; }
.text-control:hover, .icon-control:hover, .user-control:hover { color: var(--text); border-color: var(--border-strong); background: var(--surface-muted); }
.header-run:focus-visible, .text-control:focus-visible, .icon-control:focus-visible, .user-control:focus-visible { outline: 3px solid var(--focus); outline-offset: 2px; }
.archive-avatar { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 7px; color: var(--accent-ink); background: var(--accent); font: 700 13px/1 var(--font-display); }
.archive-user-name { max-width: 100px; overflow: hidden; text-overflow: ellipsis; font: 600 12px/1 var(--font-display); }
.archive-main { padding: clamp(18px, 3vw, 40px); }
.dropdown-placeholder { width: 14px; }
.archive-backdrop { position: fixed; inset: 0; z-index: 25; background: rgba(10, 12, 10, .62); backdrop-filter: blur(3px); }
.ops-route-enter-active, .ops-route-leave-active { transition: opacity .2s ease, transform .2s ease; }
.ops-route-enter-from { opacity: 0; transform: translateY(6px); }
.ops-route-leave-to { opacity: 0; transform: translateY(-4px); }
@media (max-width: 920px) {
  .archive-shell { display: block; }
  .archive-sidebar { position: fixed; left: 0; transform: translateX(-100%); width: min(86vw, 300px); transition: transform .24s ease; box-shadow: 24px 0 60px rgba(0,0,0,.28); }
  .archive-sidebar.is-open { transform: translateX(0); }
  .archive-header { position: sticky; top: 0; z-index: 20; min-height: 72px; }
  .archive-user-name, .text-control .el-icon { display: none; }
}
@media (max-width: 620px) {
  .archive-header__actions { gap: 6px; }
  .header-run, .archive-kicker, .archive-user-name { display: none; }
  .archive-header h1 { margin-top: 0; white-space: nowrap; }
  .text-control, .user-control { width: 44px; padding: 0; }
  .user-control > .el-icon { display: none; }
}
@media (prefers-reduced-motion: reduce) { .archive-sidebar, .ops-route-enter-active, .ops-route-leave-active { transition: none; } }
</style>
