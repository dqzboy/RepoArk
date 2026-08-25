<template>
  <main class="login-shell">
    <section class="login-manifest">
      <div class="login-manifest__brand"><BrandMark /><div><strong>{{ t('app.brand') }}</strong><span>{{ t('app.sub') }}</span></div></div>
      <div class="login-manifest__copy">
        <span class="login-eyebrow">{{ t('login.accessPoint') }}</span>
        <h1>{{ t('login.heroTitle') }}</h1>
        <p>{{ t('login.heroDesc') }}</p>
      </div>
      <div class="login-ledger" aria-label="Platform coverage">
        <div v-for="(platform, index) in platforms" :key="platform"><span>0{{ index + 1 }}</span><strong>{{ platform }}</strong><small>{{ t('login.archiveReady') }}</small></div>
      </div>
      <div class="login-security"><el-icon><Lock /></el-icon><span>{{ t('login.securityNote') }}</span></div>
    </section>

    <section class="login-access">
      <div class="login-card ops-fade-up">
        <div class="login-card__head"><span>{{ t('login.credentials') }}</span><strong>ARK / AUTH</strong></div>
        <h2>{{ t('login.welcome') }}</h2>
        <p>{{ t('login.formDesc') }}</p>
        <el-form :model="form" label-position="top" @submit.prevent="onSubmit">
          <el-form-item :label="t('common.username')"><el-input v-model="form.username" :placeholder="t('login.usernamePh')" :prefix-icon="User" size="large" autocomplete="username" /></el-form-item>
          <el-form-item :label="t('common.password')"><el-input v-model="form.password" type="password" :placeholder="t('login.passwordPh')" :prefix-icon="Lock" show-password size="large" autocomplete="current-password" @keyup.enter="onSubmit" /></el-form-item>
          <el-button native-type="submit" type="primary" size="large" :loading="loading" class="login-submit" @click="onSubmit">{{ t('login.submit') }}<el-icon><Right /></el-icon></el-button>
        </el-form>
        <div class="login-hint"><el-icon><InfoFilled /></el-icon><span>{{ t('login.hint') }}</span></div>
      </div>
    </section>
  </main>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { InfoFilled, Lock, Right, User } from '@element-plus/icons-vue'
import BrandMark from '../components/BrandMark.vue'
import api from '../api'
import { t } from '../i18n'

const router = useRouter()
const form = ref({ username: '', password: '' })
const loading = ref(false)
const platforms = ['GitHub', 'GitCode', 'Gitee']

async function onSubmit() {
  if (loading.value) return
  loading.value = true
  try {
    const { data } = await api.post('/auth/login', form.value)
    localStorage.setItem('token', data.token)
    localStorage.setItem('username', form.value.username)
    router.push('/')
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('login.error'))
  } finally { loading.value = false }
}
</script>

<style scoped>
.login-shell { min-height: 100vh; display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(420px, .9fr); background: var(--bg); }
.login-manifest { min-height: 100vh; display: flex; flex-direction: column; padding: clamp(28px, 6vw, 72px); color: var(--hero-text); background: var(--hero); border-right: 1px solid var(--hero-border); }
.login-manifest__brand { display: flex; align-items: center; gap: 15px; }
.login-manifest__brand strong, .login-manifest__brand span { display: block; font-family: var(--font-display); }
.login-manifest__brand strong { font-size: 14px; letter-spacing: .06em; }.login-manifest__brand span { margin-top: 5px; color: var(--hero-muted); font-size: 10px; letter-spacing: .17em; text-transform: uppercase; }
.login-manifest__copy { max-width: 680px; margin: auto 0; padding: 70px 0; }
.login-eyebrow { color: var(--accent); font: 700 10px/1 var(--font-display); letter-spacing: .22em; text-transform: uppercase; }
.login-manifest h1 { max-width: 720px; margin: 18px 0; font: 650 clamp(42px, 7vw, 88px)/.94 var(--font-body); letter-spacing: -.055em; }
.login-manifest p { max-width: 580px; color: var(--hero-muted); font-size: 15px; line-height: 1.8; }
.login-ledger { display: grid; grid-template-columns: repeat(3, 1fr); border-top: 1px solid var(--hero-border); border-bottom: 1px solid var(--hero-border); }
.login-ledger div { display: grid; gap: 7px; padding: 18px; border-right: 1px solid var(--hero-border); }.login-ledger div:last-child { border-right: 0; }
.login-ledger span, .login-ledger small { color: var(--hero-muted); font: 600 9px/1 var(--font-display); letter-spacing: .1em; text-transform: uppercase; }.login-ledger strong { font: 650 14px/1 var(--font-display); }
.login-security { display: flex; gap: 9px; align-items: center; margin-top: 20px; color: var(--hero-muted); font-size: 11px; }
.login-access { display: grid; place-items: center; padding: 32px; }
.login-card { width: min(100%, 430px); padding: clamp(25px, 5vw, 44px); background: var(--surface); border: 1px solid var(--border); border-radius: 12px 12px 12px 4px; box-shadow: 12px 12px 0 var(--shadow-line); }
.login-card__head { display: flex; justify-content: space-between; color: var(--text-faint); font: 650 9px/1 var(--font-display); letter-spacing: .16em; text-transform: uppercase; }.login-card__head strong { color: var(--accent-strong); }
.login-card h2 { margin: 36px 0 9px; color: var(--text); font-size: 28px; }.login-card > p { margin: 0 0 28px; color: var(--text-muted); font-size: 13px; }
.login-submit { width: 100%; min-height: 48px; margin-top: 7px; letter-spacing: .16em; }
.login-hint { display: flex; align-items: flex-start; gap: 9px; margin-top: 20px; padding-top: 17px; color: var(--text-faint); border-top: 1px solid var(--border); font-size: 11px; line-height: 1.6; }
@media (max-width: 900px) { .login-shell { grid-template-columns: 1fr; }.login-manifest { min-height: auto; padding-bottom: 28px; }.login-manifest__copy { padding: 70px 0 50px; }.login-manifest h1 { font-size: clamp(40px, 11vw, 70px); }.login-security { display: none; }.login-access { min-height: 580px; } }
@media (max-width: 560px) { .login-manifest { padding: 24px 18px; }.login-ledger small { display: none; }.login-ledger div { padding: 14px 8px; }.login-access { padding: 26px 18px; }.login-card { padding: 26px 20px; } }
</style>
