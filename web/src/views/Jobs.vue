<template>
  <el-card class="ops-card">
    <template #header>
      <div style="display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap">
        <span class="ops-card__title"><el-icon class="ops-card__icon"><List /></el-icon>{{ t('jobs.title') }}</span>
        <el-radio-group v-model="filterPlatform" size="small" @change="load">
          <el-radio-button value="">{{ t('jobs.allPlatforms') }}</el-radio-button>
          <el-radio-button v-for="p in PLATFORMS" :key="p.code" :value="p.code">
            {{ t('platform.' + p.code) }}
          </el-radio-button>
        </el-radio-group>
      </div>
    </template>
    <el-table :data="jobs" style="width: 100%" v-loading="loading" element-loading-background="transparent">
      <el-table-column prop="id" :label="t('jobs.id')" width="80" />
      <el-table-column :label="t('jobs.platform')" width="120">
        <template #default="{ row }">
          <span class="ops-pill" :class="platformPillClass(row.platform)">
            <i class="ops-platform-dot" :style="{ background: dotColor(row.platform) }" />
            {{ t('platform.' + (row.platform || 'github')) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column prop="server_name" :label="t('jobs.server')" width="170" />
      <el-table-column :label="t('jobs.status')" width="130">
        <template #default="{ row }">
          <span class="ops-pill" :class="pillClass(row.status)">{{ statusText(row.status) }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="message" :label="t('jobs.result')" />
      <el-table-column prop="started_at" :label="t('jobs.started')" width="180" />
      <el-table-column prop="finished_at" :label="t('jobs.finished')" width="180" />
      <el-table-column :label="t('jobs.actions')" width="100">
        <template #default="{ row }">
          <el-button link type="primary" @click="viewLog(row)">{{ t('jobs.log') }}</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <EmptyState :title="t('jobs.empty')" />
      </template>
    </el-table>

    <el-dialog v-model="dialog" :title="t('jobs.logTitle')" width="72%">
      <pre class="ops-log">{{ activeLog }}</pre>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { List } from '@element-plus/icons-vue'
import EmptyState from '../components/EmptyState.vue'
import api from '../api'
import { t } from '../i18n'
import { PLATFORMS } from '../stores/platform'

const jobs = ref([])
const dialog = ref(false)
const activeLog = ref('')
const loading = ref(true)
const filterPlatform = ref('')

function dotColor(code) {
  if (code === 'github') return '#7c5cff'
  if (code === 'gitcode') return '#22b8ff'
  if (code === 'gitee') return '#ff6b35'
  return '#888'
}
function platformPillClass(code) {
  if (code === 'github') return 'ops-pill--platform-github'
  if (code === 'gitcode') return 'ops-pill--platform-gitcode'
  if (code === 'gitee') return 'ops-pill--platform-gitee'
  return ''
}
function statusText(s) {
  return s === 'success' ? t('status.success') : s === 'failed' ? t('status.failed') : s === 'running' ? t('status.running') : t('status.unknown')
}
function pillClass(s) {
  if (s === 'success') return 'ops-pill--success'
  if (s === 'failed') return 'ops-pill--failed'
  if (s === 'running') return 'ops-pill--running'
  return 'ops-pill--idle'
}

async function load() {
  loading.value = true
  try {
    const url = filterPlatform.value ? '/jobs?platform=' + filterPlatform.value : '/jobs'
    const { data } = await api.get(url)
    jobs.value = data
  } finally {
    loading.value = false
  }
}

function viewLog(row) {
  activeLog.value = row.log || t('jobs.noLog')
  dialog.value = true
}

onMounted(load)
watch(filterPlatform, load)
</script>

<style scoped>
.ops-platform-dot {
  display: inline-block; width: 7px; height: 7px; border-radius: 50%;
  margin-right: 6px; vertical-align: 0;
  box-shadow: 0 0 5px currentColor;
}
.ops-pill {
  display: inline-flex; align-items: center;
  font-family: var(--font-display); font-size: 12px;
  padding: 3px 10px; border-radius: 999px;
}
.ops-pill--platform-github { background: rgba(124,92,255,0.14); color: #7c5cff; }
.ops-pill--platform-gitcode { background: rgba(34,184,255,0.14); color: #22b8ff; }
.ops-pill--platform-gitee { background: rgba(255,107,53,0.16); color: #ff6b35; }

.ops-pill--success { background: color-mix(in srgb, var(--accent) 18%, transparent); color: var(--accent); }
.ops-pill--failed  { background: color-mix(in srgb, var(--danger) 18%, transparent); color: var(--danger); }
.ops-pill--running { background: color-mix(in srgb, var(--warn) 18%, transparent); color: var(--warn); }
.ops-pill--idle    { background: var(--surface-2); color: var(--text-muted); }

.ops-log {
  margin: 0;
  padding: 14px;
  background: #0d1117;
  color: #e6edf3;
  border-radius: 8px;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.5;
  max-height: 60vh;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
