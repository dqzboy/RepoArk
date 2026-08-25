<template>
  <el-card class="ops-card">
    <template #header>
      <div class="ops-jobs-head">
        <span class="ops-card__title"><el-icon class="ops-card__icon"><List /></el-icon>{{ t('jobs.title') }}</span>
        <el-radio-group v-model="filterPlatform" size="small" class="ops-jobs-filter">
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
          <span class="ops-pill ops-pill--platform">
            <span class="ops-platform-index">{{ platformIndex(row.platform) }}</span>
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

    <el-dialog v-model="dialog" :title="t('jobs.logTitle')" width="min(900px, calc(100vw - 32px))">
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

function platformIndex(code) {
  const index = PLATFORMS.findIndex((platform) => platform.code === code)
  return String(index >= 0 ? index + 1 : 0).padStart(2, '0')
}
function statusText(s) {
  if (s === 'success') return t('status.success')
  if (s === 'failed') return t('status.failed')
  if (s === 'running') return t('status.running')
  if (s === 'cancelling') return t('status.cancelling')
  if (s === 'cancelled') return t('status.cancelled')
  return t('status.unknown')
}
function pillClass(s) {
  if (s === 'success') return 'ops-pill--success'
  if (s === 'failed') return 'ops-pill--failed'
  if (s === 'running') return 'ops-pill--running'
  if (s === 'cancelling') return 'ops-pill--cancelling'
  if (s === 'cancelled') return 'ops-pill--cancelled'
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
.ops-jobs-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.ops-jobs-filter {
  max-width: 100%;
  overflow-x: auto;
}
.ops-pill--platform {
  color: var(--text-muted);
  background: var(--surface-muted);
  border: 1px solid var(--border);
}
.ops-platform-index {
  color: var(--accent-strong);
  font: 700 9px/1 var(--font-display);
  letter-spacing: .05em;
}
@media (max-width: 720px) {
  .ops-jobs-head,
  .ops-jobs-filter {
    width: 100%;
  }
}
</style>
