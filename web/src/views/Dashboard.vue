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

    <el-row :gutter="16">
      <el-col :xs="12" :sm="12" :md="6">
        <el-card class="ops-card ops-fade-up" style="animation-delay: 0.04s">
          <div class="ops-stat-label">{{ t('dashboard.repo') }}</div>
          <div class="ops-stat-value">{{ cfg.repo_name || '-' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :md="6">
        <el-card class="ops-card ops-fade-up" style="animation-delay: 0.1s">
          <div class="ops-stat-label">{{ t('dashboard.branch') }}</div>
          <div class="ops-stat-value">{{ cfg.branch || '-' }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :md="6">
        <el-card class="ops-card ops-fade-up" style="animation-delay: 0.16s">
          <div class="ops-stat-label">{{ t('dashboard.sources') }}</div>
          <div class="ops-stat-value">{{ cfg.backup_sources?.length || 0 }}</div>
        </el-card>
      </el-col>
      <el-col :xs="12" :sm="12" :md="6">
        <el-card class="ops-card ops-fade-up" style="animation-delay: 0.22s">
          <div class="ops-stat-label">{{ t('dashboard.lastJob') }}</div>
          <div class="ops-stat-value">
            <span class="ops-pill" :class="pillClass(lastJob?.status)">
              {{ lastJob ? statusText(lastJob.status) : t('common.none') }}
            </span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :xs="24" :md="10">
        <el-card class="ops-card ops-fade-up" style="animation-delay: 0.28s">
          <template #header>
            <span class="ops-card__title"><el-icon class="ops-card__icon"><PieChart /></el-icon>{{ t('dashboard.distTitle') }}</span>
          </template>
          <div v-if="total" class="ops-donut-wrap">
            <svg class="ops-donut" viewBox="0 0 120 120">
              <circle cx="60" cy="60" r="52" fill="none" stroke="var(--surface-2)" stroke-width="14" />
              <circle
                v-for="(seg, i) in donutSegments"
                :key="i"
                cx="60" cy="60" r="52" fill="none"
                :stroke="seg.color" stroke-width="14" stroke-linecap="round"
                :stroke-dasharray="`${seg.dash} ${seg.gap}`"
                :stroke-dashoffset="seg.offset"
                transform="rotate(-90 60 60)"
              />
            </svg>
            <div class="ops-donut-center">
              <div class="ops-donut-total">{{ total }}</div>
              <div class="ops-donut-cap">{{ t('dashboard.totalJobs') }}</div>
            </div>
          </div>
          <EmptyState v-else :title="t('dashboard.noJobs')" />
          <div v-if="total" class="ops-legend">
            <span class="ops-legend__item"><i class="ops-legend__dot" style="background: var(--accent)" />{{ t('status.success') }} · {{ statusCounts.success }}</span>
            <span class="ops-legend__item"><i class="ops-legend__dot" style="background: var(--danger)" />{{ t('status.failed') }} · {{ statusCounts.failed }}</span>
            <span class="ops-legend__item"><i class="ops-legend__dot" style="background: var(--warn)" />{{ t('status.running') }} · {{ statusCounts.running }}</span>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :md="14">
        <el-card class="ops-card ops-fade-up" style="animation-delay: 0.34s">
          <template #header>
            <span class="ops-card__title"><el-icon class="ops-card__icon"><Histogram /></el-icon>{{ t('dashboard.recentTitle') }}</span>
          </template>
          <div v-if="recentBars.length" class="ops-bars">
            <div
              v-for="(b, i) in recentBars"
              :key="i"
              class="ops-bar"
              :style="{ height: b.h + '%', background: b.color }"
              :title="b.title"
            />
          </div>
          <EmptyState v-else :title="t('dashboard.noJobs')" />
        </el-card>
      </el-col>
    </el-row>

    <el-card class="ops-card ops-fade-up" style="margin-top: 16px; animation-delay: 0.4s">
      <template #header>
        <span class="ops-card__title">
          <el-icon class="ops-card__icon"><DataLine /></el-icon>{{ t('dashboard.overview') }}
          <el-tag size="small" style="margin-left: 8px">{{ platform }}</el-tag>
        </span>
      </template>
      <el-descriptions :column="2" border>
        <el-descriptions-item :label="t('dashboard.gitUser', { platform })">{{ cfg.git_user || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.repoName')">{{ cfg.repo_name || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.branchL')">{{ cfg.branch || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.backupDir')">{{ cfg.backup_dir || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.serverId')">{{ cfg.server_name || t('dashboard.autoIp') }}</el-descriptions-item>
        <el-descriptions-item :label="t('schedule.cronExpr')">
          {{ cfg.schedule_cron || '-' }}
          <span v-if="cfg.schedule_enabled" class="ops-muted">· {{ t('schedule.on') }}</span>
        </el-descriptions-item>
        <el-descriptions-item :label="t('dashboard.sourcesL')" :span="2">
          <el-tag v-for="s in cfg.backup_sources" :key="s" style="margin-right: 6px; margin-bottom: 4px">{{ s }}</el-tag>
          <span v-if="!cfg.backup_sources?.length" style="color: var(--text-muted)">{{ t('dashboard.notConfigured') }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <el-alert
        v-if="!cfg.enabled"
        type="info"
        :title="t('platform.disabledHint')"
        :closable="false"
        style="margin-top: 14px"
      />
      <el-alert
        v-else-if="!cfg.git_token"
        type="warning"
        :title="t('dashboard.tokenWarn', { platform })"
        :closable="false"
        style="margin-top: 14px"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { DataLine, PieChart, Histogram } from '@element-plus/icons-vue'
import EmptyState from '../components/EmptyState.vue'
import api from '../api'
import { t } from '../i18n'
import { PLATFORMS, usePlatformStore, platformLabel as pLabel } from '../stores/platform'

const store = usePlatformStore()
const active = ref(store.platform)
const platform = computed(() => pLabel(active.value))

const cfg = ref({})
const jobs = ref([])
const lastJob = ref(null)
const loading = ref(true)
const enabledMap = ref({})

function dotColor(code) {
  if (code === 'github') return '#7c5cff'
  if (code === 'gitcode') return '#22b8ff'
  if (code === 'gitee') return '#ff6b35'
  return '#888'
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
function colorOf(s) {
  if (s === 'success') return 'var(--accent)'
  if (s === 'failed') return 'var(--danger)'
  if (s === 'running') return 'var(--warn)'
  return 'var(--text-muted)'
}

const total = computed(() => jobs.value.length)
const statusCounts = computed(() => {
  const c = { success: 0, failed: 0, running: 0 }
  for (const j of jobs.value) {
    if (j.status === 'success') c.success++
    else if (j.status === 'failed') c.failed++
    else if (j.status === 'running') c.running++
  }
  return c
})

const donutSegments = computed(() => {
  const r = 52
  const C = 2 * Math.PI * r
  const order = [
    { key: 'success', color: 'var(--accent)' },
    { key: 'failed', color: 'var(--danger)' },
    { key: 'running', color: 'var(--warn)' }
  ]
  const segs = []
  let acc = 0
  for (const o of order) {
    const frac = total.value ? statusCounts.value[o.key] / total.value : 0
    const dash = frac * C
    segs.push({ color: o.color, dash, gap: C - dash, offset: -acc })
    acc += dash
  }
  return segs
})

const recentBars = computed(() => {
  const list = jobs.value.slice(0, 16)
  return list.map((j) => {
    let h = 55
    if (j.status === 'success') h = 100
    else if (j.status === 'running') h = 86
    else if (j.status === 'failed') h = 68
    return { h, color: colorOf(j.status), title: `${j.id} · ${statusText(j.status)}` }
  })
})

async function loadOne(code) {
  try {
    const c = await api.get('/platforms/' + code)
    cfg.value = c.data
  } catch (e) {
    cfg.value = {}
  }
  try {
    const j = await api.get('/jobs?platform=' + code)
    jobs.value = j.data || []
    lastJob.value = jobs.value[0] || null
  } catch (e) {
    jobs.value = []
    lastJob.value = null
  }
}

async function loadEnabledMap() {
  try {
    const { data } = await api.get('/platforms')
    const m = {}
    for (const p of data || []) m[p.platform] = !!p.enabled
    enabledMap.value = m
  } catch (_) {}
}

async function onTabChange(tab) {
  store.platform = tab
  await loadOne(tab)
}

onMounted(async () => {
  await loadEnabledMap()
  await loadOne(active.value)
  loading.value = false
})
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
.ops-fade-up { animation: ops-fade-up 0.36s cubic-bezier(0.2, 0, 0, 1) both; }
@keyframes ops-fade-up {
  from { opacity: 0; transform: translateY(8px); }
  to   { opacity: 1; transform: translateY(0); }
}

.ops-stat-label {
  font-family: var(--font-display); font-size: 11px; letter-spacing: 0.18em;
  text-transform: uppercase; color: var(--text-muted);
}
.ops-stat-value {
  font-family: var(--font-display); font-size: 22px; font-weight: 600;
  color: var(--text); margin-top: 6px;
}
.ops-pill { display: inline-flex; align-items: center; font-family: var(--font-display); font-size: 12px; padding: 3px 10px; border-radius: 999px; }
.ops-pill--success { background: color-mix(in srgb, var(--accent) 18%, transparent); color: var(--accent); }
.ops-pill--failed  { background: color-mix(in srgb, var(--danger) 18%, transparent); color: var(--danger); }
.ops-pill--running { background: color-mix(in srgb, var(--warn) 18%, transparent); color: var(--warn); }
.ops-pill--idle    { background: var(--surface-2); color: var(--text-muted); }

.ops-donut-wrap {
  position: relative;
  width: 120px;
  height: 120px;
  margin: 6px auto 4px;
}
.ops-donut { width: 120px; height: 120px; }
.ops-donut-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}
.ops-donut-total {
  font-family: var(--font-display);
  font-size: 26px;
  font-weight: 700;
  color: var(--text);
  line-height: 1;
}
.ops-donut-cap {
  font-family: var(--font-display);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-muted);
  margin-top: 4px;
}
.ops-legend { display: flex; flex-wrap: wrap; gap: 10px 18px; justify-content: center; margin-top: 12px; font-family: var(--font-display); font-size: 12px; color: var(--text-muted); }
.ops-legend__item { display: inline-flex; align-items: center; gap: 7px; }
.ops-legend__dot { width: 9px; height: 9px; border-radius: 3px; display: inline-block; }
.ops-bars { display: flex; align-items: flex-end; gap: 6px; height: 132px; padding: 8px 2px 0; }
.ops-bar { flex: 1; min-width: 6px; border-radius: 5px 5px 2px 2px; opacity: 0.85; transition: opacity 0.2s ease, transform 0.2s ease; }
.ops-bar:hover { opacity: 1; transform: translateY(-2px); }
.ops-muted { color: var(--text-muted); font-family: var(--font-display); font-size: 12px; margin-left: 6px; }
</style>
