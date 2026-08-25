<template>
  <div v-loading="loading" element-loading-background="transparent">
    <PlatformSwitcher v-model="active" :enabled-map="enabledMap" @change="onTabChange" />

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
          <div class="ops-result-icon"><el-icon><CircleCheck v-if="currentJob.status === 'success'" /><CircleClose v-else-if="currentJob.status === 'failed'" /><Loading v-else class="backup-spinner" /></el-icon></div>
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
import { CircleCheck, CircleClose, Loading, VideoPlay, Upload } from '@element-plus/icons-vue'
import PlatformSwitcher from '../components/PlatformSwitcher.vue'
import api from '../api'
import { t } from '../i18n'
import { usePlatformStore, platformLabel as pLabel } from '../stores/platform'

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

function statusText(s) {
  return s === 'success' ? t('backup.success') : s === 'failed' ? t('backup.failed') : s === 'running' ? t('backup.running') : t('backup.unknown')
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
    poll(data.id)
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('backup.startError'))
    running.value = false
  }
}

function poll(id) {
  stopPoll()
  pollJobId = id
  pollPlatform = active.value
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
.backup-spinner { animation: ops-spin 1s linear infinite; }
</style>
