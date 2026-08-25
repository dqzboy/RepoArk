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
          <div class="backup-actions">
            <el-button
              v-if="running"
              type="danger"
              plain
              :icon="CircleClose"
              :loading="cancelRequested || currentJob?.status === 'cancelling'"
              :disabled="!currentJob?.id"
              @click="cancelBackup"
            >
              {{ currentJob?.status === 'cancelling' ? t('backup.cancelling') : t('backup.cancel') }}
            </el-button>
            <el-button v-else type="primary" :icon="VideoPlay" :disabled="!enabledMap[active]" @click="run">
              {{ t('backup.start') }}
            </el-button>
          </div>
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
          <div class="ops-result-icon"><el-icon><CircleCheck v-if="currentJob.status === 'success'" /><CircleClose v-else-if="['failed', 'cancelled'].includes(currentJob.status)" /><Loading v-else class="backup-spinner" /></el-icon></div>
          <div class="backup-result-content">
            <div class="ops-result-title">{{ statusText(currentJob.status) }}</div>
            <div class="ops-result-sub">{{ currentJob.message || t('backup.runningMsg') }}</div>
            <div class="ops-result-meta">{{ t('backup.jobMeta', { id: currentJob.id, server: currentJob.server_name }) }}</div>
          </div>
        </div>
        <div v-if="isJobActive" class="backup-progress" aria-live="polite">
          <div class="backup-progress__head">
            <span><Loading class="backup-progress__phase-icon" />{{ phaseText }}</span>
            <strong>{{ jobProgress }}%</strong>
          </div>
          <el-progress
            :percentage="jobProgress"
            :stroke-width="10"
            :show-text="false"
            striped
            striped-flow
          />
        </div>
        <pre class="ops-log">{{ currentJob.log }}</pre>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
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
const cancelRequested = ref(false)
let timer = null
let pollJobId = null
let pollPlatform = null
let pollVersion = 0
let viewVersion = 0

function statusText(s) {
  if (s === 'success') return t('backup.success')
  if (s === 'failed') return t('backup.failed')
  if (s === 'running') return t('backup.running')
  if (s === 'cancelling') return t('backup.cancelling')
  if (s === 'cancelled') return t('backup.cancelled')
  return t('backup.unknown')
}

const isJobActive = computed(() => ['running', 'cancelling'].includes(currentJob.value?.status))
const jobProgress = computed(() => Math.max(0, Math.min(100, Number(currentJob.value?.progress) || 0)))
const phaseText = computed(() => {
  const phases = {
    preparing: 'phasePreparing',
    scanning: 'phaseScanning',
    connecting: 'phaseConnecting',
    fetching: 'phaseFetching',
    packing: 'phasePacking',
    committing: 'phaseCommitting',
    pushing: 'phasePushing',
    retrying: 'phaseRetrying',
    cancelling: 'phaseCancelling'
  }
  return t('backup.' + (phases[currentJob.value?.phase] || 'phasePreparing'))
})

async function loadEnabledMap() {
  const { data } = await api.get('/platforms')
  const m = {}
  for (const p of data || []) m[p.platform] = !!p.enabled
  enabledMap.value = m
}

async function onTabChange(tab) {
  store.platform = tab
  await restoreRunningJob(tab)
}

async function init() {
  loading.value = true
  try {
    await loadEnabledMap()
    await restoreRunningJob(active.value)
  } finally {
    loading.value = false
  }
}

// 页面重新进入或切换平台时，从后端恢复该平台正在运行的任务。
async function restoreRunningJob(platformCode) {
  const version = ++viewVersion
  stopPoll()
  running.value = false
  currentJob.value = null
  try {
    const { data } = await api.get('/jobs?platform=' + encodeURIComponent(platformCode))
    if (version !== viewVersion || platformCode !== active.value) return
    const job = (data || []).find((item) => ['running', 'cancelling'].includes(item.status))
    if (!job) return
    currentJob.value = job
    running.value = true
    poll(job.id, platformCode)
  } catch (e) {
    /* 恢复任务失败时保持初始状态，用户仍可正常重试 */
  }
}

async function run() {
  const platformCode = active.value
  const version = ++viewVersion
  stopPoll()
  running.value = true
  currentJob.value = null
  try {
    const { data } = await api.post('/backup/run/' + platformCode)
    if (version !== viewVersion || platformCode !== active.value) return
    currentJob.value = {
      id: data.id,
      platform: platformCode,
      status: 'running',
      phase: 'preparing',
      progress: 1,
      server_name: '—',
      message: t('backup.runningMsg'),
      log: ''
    }
    poll(data.id, platformCode)
  } catch (e) {
    if (version !== viewVersion || platformCode !== active.value) return
    ElMessage.error(e.response?.data?.error || t('backup.startError'))
    running.value = false
  }
}

async function cancelBackup() {
  const job = currentJob.value
  if (!job?.id || !['running', 'cancelling'].includes(job.status) || cancelRequested.value) return
  try {
    await ElMessageBox.confirm(t('backup.cancelConfirm'), t('backup.cancelTitle'), {
      type: 'warning',
      confirmButtonText: t('backup.cancel'),
      cancelButtonText: t('common.cancel')
    })
  } catch {
    return
  }

  cancelRequested.value = true
  try {
    await api.post('/backup/cancel/' + job.id)
    if (currentJob.value?.id === job.id) {
      currentJob.value = {
        ...currentJob.value,
        status: 'cancelling',
        phase: 'cancelling',
        message: t('backup.cancellingMsg')
      }
    }
    ElMessage.info(t('backup.cancelRequested'))
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('backup.cancelError'))
  } finally {
    cancelRequested.value = false
  }
}

function poll(id, platformCode) {
  stopPoll()
  const version = pollVersion
  pollJobId = id
  pollPlatform = platformCode
  let refreshing = false

  const refresh = async () => {
    if (refreshing) return
    // 已被新的轮询替代时直接退出，不能反过来停止新轮询。
    if (version !== pollVersion) return
    if (pollPlatform !== active.value || pollJobId !== id) {
      // 切到别的平台后停止轮询
      stopPoll()
      return
    }
    refreshing = true
    try {
      const { data } = await api.get('/jobs/' + id)
      if (version !== pollVersion || pollPlatform !== platformCode || pollJobId !== id || active.value !== platformCode) return
      currentJob.value = data
      if (!['running', 'cancelling'].includes(data.status)) {
        running.value = false
        stopPoll()
        if (data.status === 'success') ElMessage.success(t('backup.doneSuccess'))
        else if (data.status === 'failed') ElMessage.error(t('backup.doneFailed'))
        else if (data.status === 'cancelled') ElMessage.warning(t('backup.doneCancelled'))
      }
    } catch (e) {
      /* 忽略轮询错误 */
    } finally {
      refreshing = false
    }
  }

  // 立即查询一次，避免启动或恢复任务后等待一个轮询周期才显示。
  void refresh()
  timer = setInterval(() => void refresh(), 1000)
}

function stopPoll() {
  pollVersion++
  if (timer) { clearInterval(timer); timer = null }
  pollJobId = null
  pollPlatform = null
}

function cleanup() {
  viewVersion++
  stopPoll()
}

init()
onUnmounted(cleanup)
</script>

<style scoped>
.backup-spinner { animation: ops-spin 1s linear infinite; }
.backup-actions { display: flex; align-items: center; gap: 10px; }
.backup-result-content { min-width: 0; }
.backup-progress { margin: 14px 0 0; padding: 14px 16px; background: var(--surface-muted); border: 1px solid var(--border); border-radius: 8px; }
.backup-progress__head { display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-bottom: 10px; color: var(--text-muted); font-size: 12px; }
.backup-progress__head span { min-width: 0; display: inline-flex; align-items: center; gap: 8px; }
.backup-progress__head strong { color: var(--text); font: 700 12px/1 var(--font-display); }
.backup-progress__phase-icon { width: 15px; flex: 0 0 15px; animation: ops-spin 1s linear infinite; }
</style>
