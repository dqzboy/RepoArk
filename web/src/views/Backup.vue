<template>
  <div v-loading="loading" element-loading-background="transparent">
    <el-tabs v-model="active" class="ops-platform-tabs" @tab-change="onTabChange">
      <el-tab-pane v-for="p in PLATFORMS" :key="p.code" :name="p.code">
        <template #label>
          <span class="ops-platform-tab">
            <i class="ops-platform-dot" :style="{ background: dotColor(p.code) }" :class="{ 'ops-platform-dot--off': !enabledMap[p.code] }" />
            {{ t('platform.' + p.code) }}
          </span>
        </template>
      </el-tab-pane>
    </el-tabs>

    <el-card class="ops-card">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span class="ops-card__title">
            <el-icon class="ops-card__icon"><Upload /></el-icon>{{ t('backup.title') }}
            <el-tag size="small" style="margin-left: 8px">{{ platform }}</el-tag>
          </span>
          <el-button type="primary" :icon="VideoPlay" :loading="running" :disabled="!enabledMap[active]" @click="run">
            {{ t('backup.start') }}
          </el-button>
        </div>
      </template>

      <el-alert
        v-if="!enabledMap[active]"
        type="warning"
        :closable="false"
        :title="t('backup.notEnabled')"
        style="margin-bottom: 14px"
      />
      <el-alert
        v-else-if="!currentJob"
        type="info"
        :title="t('backup.info', { platform })"
        :closable="false"
      />

      <div v-if="currentJob">
        <div class="ops-result" :class="'ops-result--' + currentJob.status">
          <div class="ops-result-icon">{{ resultIcon(currentJob.status) }}</div>
          <div>
            <div class="ops-result-title">{{ statusText(currentJob.status) }}</div>
            <div class="ops-result-sub">{{ currentJob.message || t('backup.runningMsg') }}</div>
            <div class="ops-result-meta">{{ t('backup.jobMeta', { id: currentJob.id, server: currentJob.server_name }) }}</div>
          </div>
        </div>
        <pre class="ops-log">{{ currentJob.log }}</pre>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { VideoPlay, Upload } from '@element-plus/icons-vue'
import api from '../api'
import { t } from '../i18n'
import { PLATFORMS, usePlatformStore, platformLabel as pLabel } from '../stores/platform'

const store = usePlatformStore()
const active = ref(store.platform)
const platform = computed(() => pLabel(active.value))

const running = ref(false)
const enabledMap = ref({})
const currentJob = ref(null)
const loading = ref(false)
let timer = null
let pollJobId = null
let pollPlatform = null

function dotColor(code) {
  if (code === 'github') return '#7c5cff'
  if (code === 'gitcode') return '#22b8ff'
  return '#ff6b35'
}

function statusText(s) {
  return s === 'success' ? t('backup.success') : s === 'failed' ? t('backup.failed') : s === 'running' ? t('backup.running') : t('backup.unknown')
}
function resultIcon(s) {
  return s === 'success' ? '✓' : s === 'failed' ? '✕' : '◌'
}

async function loadEnabledMap() {
  const { data } = await api.get('/platforms')
  const m = {}
  for (const p of data || []) m[p.platform] = !!p.enabled
  enabledMap.value = m
}

async function onTabChange(tab) {
  store.platform = tab
  // 切平台时清空当前 job 显示，避免跨平台串台
  stopPoll()
  currentJob.value = null
}

async function init() {
  loading.value = true
  try {
    await loadEnabledMap()
  } finally {
    loading.value = false
  }
}

async function run() {
  running.value = true
  try {
    const { data } = await api.post('/backup/run/' + active.value)
    pollJobId = data.id
    pollPlatform = active.value
    poll(data.id)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('backup.startError'))
    running.value = false
  }
}

function poll(id) {
  stopPoll()
  timer = setInterval(async () => {
    if (pollPlatform !== active.value) {
      // 切到别的平台后停止轮询
      stopPoll()
      return
    }
    try {
      const { data } = await api.get('/jobs/' + id)
      currentJob.value = data
      if (data.status !== 'running') {
        running.value = false
        stopPoll()
        if (data.status === 'success') ElMessage.success(t('backup.doneSuccess'))
        else if (data.status === 'failed') ElMessage.error(t('backup.doneFailed'))
      }
    } catch (e) {
      /* 忽略轮询错误 */
    }
  }, 1000)
}

function stopPoll() {
  if (timer) { clearInterval(timer); timer = null }
  pollJobId = null
  pollPlatform = null
}

init()
onUnmounted(stopPoll)
</script>

<style scoped>
.ops-platform-tabs { margin-bottom: 14px; }
.ops-platform-tabs :deep(.el-tabs__nav-wrap::after) { background-color: transparent; }
.ops-platform-tab { display: inline-flex; align-items: center; gap: 8px; padding: 4px 10px; }
.ops-platform-dot {
  display: inline-block; width: 8px; height: 8px; border-radius: 50%;
  box-shadow: 0 0 6px currentColor; transition: opacity 0.2s ease;
}
.ops-platform-dot--off { opacity: 0.35; box-shadow: none; }
.ops-result {
  display: flex; align-items: flex-start; gap: 14px;
  padding: 16px; border-radius: 10px;
  background: var(--surface-2); border: 1px solid var(--border);
}
.ops-result--success { border-color: color-mix(in srgb, var(--accent) 40%, transparent); background: color-mix(in srgb, var(--accent) 10%, var(--surface-2)); }
.ops-result--failed  { border-color: color-mix(in srgb, var(--danger) 40%, transparent); background: color-mix(in srgb, var(--danger) 10%, var(--surface-2)); }
.ops-result--running { border-color: color-mix(in srgb, var(--warn) 40%, transparent); background: color-mix(in srgb, var(--warn) 10%, var(--surface-2)); }
.ops-result-icon {
  width: 38px; height: 38px; border-radius: 8px;
  display: flex; align-items: center; justify-content: center;
  font-size: 20px; font-weight: 700; color: var(--surface);
}
.ops-result--success .ops-result-icon { background: var(--accent); }
.ops-result--failed  .ops-result-icon { background: var(--danger); }
.ops-result--running .ops-result-icon { background: var(--warn); color: #1a1a1a; }
.ops-result-title { font-family: var(--font-display); font-weight: 700; font-size: 16px; color: var(--text); }
.ops-result-sub { font-family: var(--font-display); font-size: 13px; color: var(--text-muted); margin-top: 4px; }
.ops-result-meta { font-family: var(--font-display); font-size: 11px; color: var(--text-muted); margin-top: 8px; letter-spacing: 0.06em; }
.ops-log {
  margin-top: 14px;
  padding: 14px;
  background: #0d1117;
  color: #e6edf3;
  border-radius: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.5;
  max-height: 420px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
