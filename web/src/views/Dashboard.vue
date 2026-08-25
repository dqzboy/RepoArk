<template>
  <div v-loading="loading" class="control-dashboard" element-loading-background="transparent">
    <section class="dashboard-hero ops-fade-up">
      <div class="dashboard-hero__copy">
        <span class="eyebrow">{{ t('dashboard.commandCenter') }}</span>
        <h2>{{ t('dashboard.heroTitle') }}</h2>
        <p>{{ t('dashboard.heroDesc') }}</p>
        <div class="dashboard-hero__actions">
          <router-link class="dashboard-primary-action" to="/backup"><el-icon><VideoPlay /></el-icon>{{ t('dashboard.runNow') }}</router-link>
          <router-link class="dashboard-secondary-action" to="/config"><el-icon><Setting /></el-icon>{{ t('dashboard.reviewConfig') }}</router-link>
        </div>
      </div>
      <div class="dashboard-hero__stamp" aria-hidden="true">
        <span>{{ t('dashboard.archiveHealth') }}</span>
        <strong>{{ healthScore }}%</strong>
        <small>{{ healthLabel }}</small>
      </div>
    </section>

    <section class="metric-strip" :aria-label="t('dashboard.metrics')">
      <article v-for="metric in metrics" :key="metric.label" class="metric-cell ops-fade-up">
        <span>{{ metric.index }}</span><div><small>{{ metric.label }}</small><strong>{{ metric.value }}</strong><em>{{ metric.note }}</em></div>
      </article>
    </section>

    <section class="dashboard-section">
      <div class="section-heading"><div><span class="eyebrow">{{ t('dashboard.platformMatrix') }}</span><h3>{{ t('dashboard.threePlatformTitle') }}</h3></div><router-link to="/config">{{ t('dashboard.managePlatforms') }}<el-icon><ArrowRight /></el-icon></router-link></div>
      <div class="platform-grid">
        <article v-for="(platform, index) in platformCards" :key="platform.code" class="platform-card ops-fade-up" :style="{ '--delay': `${index * 60}ms` }">
          <div class="platform-card__top">
            <span class="platform-card__number">0{{ index + 1 }}</span>
            <span class="platform-card__status" :class="{ 'is-off': !platform.enabled }"><i />{{ platform.enabled ? t('platform.ready') : t('platform.offline') }}</span>
          </div>
          <div class="platform-card__brand">
            <span class="platform-glyph"><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path :d="platform.icon" /></svg></span>
            <div><h4>{{ platform.label }}</h4><p>{{ platform.git_user || t('dashboard.notConfigured') }}</p></div>
          </div>
          <dl class="platform-card__facts">
            <div><dt>{{ t('dashboard.repo') }}</dt><dd>{{ platform.repo_name || '—' }}</dd></div>
            <div><dt>{{ t('dashboard.branch') }}</dt><dd>{{ platform.branch || '—' }}</dd></div>
            <div><dt>{{ t('dashboard.sources') }}</dt><dd>{{ platform.backup_sources?.length || 0 }}</dd></div>
          </dl>
          <div class="platform-card__last">
            <span>{{ t('dashboard.lastJob') }}</span>
            <strong class="status-badge" :class="statusClass(platform.lastJob?.status)">{{ platform.lastJob ? statusText(platform.lastJob.status) : t('dashboard.noRuns') }}</strong>
          </div>
          <div class="platform-card__actions"><router-link to="/backup" @click="setPlatform(platform.code)">{{ t('dashboard.runBackup') }}</router-link><router-link to="/config" @click="setPlatform(platform.code)">{{ t('common.edit') }}</router-link></div>
        </article>
      </div>
    </section>

    <section class="dashboard-lower">
      <article class="activity-panel ops-panel">
        <div class="panel-heading"><div><span class="eyebrow">{{ t('dashboard.activityIndex') }}</span><h3>{{ t('dashboard.recentTitle') }}</h3></div><router-link to="/jobs">{{ t('dashboard.viewAll') }}</router-link></div>
        <div v-if="recentJobs.length" class="activity-list">
          <div v-for="job in recentJobs" :key="job.id" class="activity-row">
            <span class="activity-row__line" :class="statusClass(job.status)" />
            <span class="activity-row__id">#{{ String(job.id).padStart(4, '0') }}</span>
            <div class="activity-row__main"><strong>{{ platformName(job.platform) }}</strong><span>{{ job.message || statusText(job.status) }}</span></div>
            <time>{{ job.started_at || '—' }}</time>
            <span class="status-badge" :class="statusClass(job.status)">{{ statusText(job.status) }}</span>
          </div>
        </div>
        <EmptyState v-else :title="t('dashboard.noJobs')" :desc="t('dashboard.noJobsDesc')"><router-link class="inline-action" to="/backup">{{ t('dashboard.runNow') }}</router-link></EmptyState>
      </article>

      <aside class="status-panel ops-panel">
        <div class="panel-heading"><div><span class="eyebrow">{{ t('dashboard.jobQuality') }}</span><h3>{{ t('dashboard.distTitle') }}</h3></div></div>
        <div class="status-panel__rate"><strong>{{ successRate }}%</strong><span>{{ t('dashboard.successRate') }}</span></div>
        <div class="quality-bar" :aria-label="t('dashboard.distributionSummary', statusCounts)"><i class="is-success" :style="{ width: `${percentOf(statusCounts.success)}%` }" /><i class="is-failed" :style="{ width: `${percentOf(statusCounts.failed)}%` }" /><i class="is-running" :style="{ width: `${percentOf(statusCounts.running)}%` }" /><i class="is-cancelled" :style="{ width: `${percentOf(statusCounts.cancelled)}%` }" /></div>
        <dl class="quality-list">
          <div><dt><i class="is-success" />{{ t('status.success') }}</dt><dd>{{ statusCounts.success }}</dd></div>
          <div><dt><i class="is-failed" />{{ t('status.failed') }}</dt><dd>{{ statusCounts.failed }}</dd></div>
          <div><dt><i class="is-running" />{{ t('status.running') }}</dt><dd>{{ statusCounts.running }}</dd></div>
          <div><dt><i class="is-cancelled" />{{ t('status.cancelled') }}</dt><dd>{{ statusCounts.cancelled }}</dd></div>
        </dl>
        <div class="status-panel__note"><el-icon><Lock /></el-icon><span>{{ t('dashboard.credentialNote') }}</span></div>
      </aside>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, Lock, Setting, VideoPlay } from '@element-plus/icons-vue'
import EmptyState from '../components/EmptyState.vue'
import api from '../api'
import { t } from '../i18n'
import { PLATFORMS, platformLabel, usePlatformStore } from '../stores/platform'

const store = usePlatformStore()
const profiles = ref([])
const jobs = ref([])
const loading = ref(true)

const platformCards = computed(() => PLATFORMS.map((meta) => {
  const profile = profiles.value.find((item) => item.platform === meta.code) || {}
  return { ...meta, ...profile, code: meta.code, label: platformLabel(meta.code), lastJob: jobs.value.find((job) => job.platform === meta.code) }
}))
const enabledCount = computed(() => profiles.value.filter((item) => item.enabled).length)
const statusCounts = computed(() => jobs.value.reduce((count, job) => {
  if (job.status === 'cancelling') {
    count.running += 1
    return count
  }
  if (Object.hasOwn(count, job.status)) count[job.status] += 1
  return count
}, { success: 0, failed: 0, running: 0, cancelled: 0 }))
const completedCount = computed(() => statusCounts.value.success + statusCounts.value.failed)
const successRate = computed(() => completedCount.value ? Math.round((statusCounts.value.success / completedCount.value) * 100) : 0)
const healthScore = computed(() => Math.round((enabledCount.value / 3) * 45 + (successRate.value / 100) * 55))
const healthLabel = computed(() => healthScore.value >= 85 ? t('dashboard.healthGood') : healthScore.value >= 55 ? t('dashboard.healthWatch') : t('dashboard.healthSetup'))
const recentJobs = computed(() => jobs.value.slice(0, 8))
const metrics = computed(() => [
  { index: '01', label: t('dashboard.enabledPlatforms'), value: `${enabledCount.value} / 3`, note: t('dashboard.enabledNote') },
  { index: '02', label: t('dashboard.successRate'), value: `${successRate.value}%`, note: t('dashboard.completedJobs', { count: completedCount.value }) },
  { index: '03', label: t('dashboard.runningJobs'), value: statusCounts.value.running, note: t('dashboard.liveQueue') },
  { index: '04', label: t('dashboard.totalJobs'), value: jobs.value.length, note: t('dashboard.latestHundred') }
])

function platformName(code) { return platformLabel(code || 'github') }
function statusText(status) {
  if (status === 'success') return t('status.success')
  if (status === 'failed') return t('status.failed')
  if (status === 'running') return t('status.running')
  if (status === 'cancelling') return t('status.cancelling')
  if (status === 'cancelled') return t('status.cancelled')
  return t('status.unknown')
}
function statusClass(status) { return `is-${status || 'idle'}` }
function percentOf(value) { return jobs.value.length ? (value / jobs.value.length) * 100 : 0 }
function setPlatform(code) { store.platform = code }

onMounted(async () => {
  try {
    const [platformResponse, jobResponse] = await Promise.all([api.get('/platforms'), api.get('/jobs')])
    profiles.value = platformResponse.data || []
    jobs.value = jobResponse.data || []
  } finally { loading.value = false }
})
</script>

<style scoped>
.control-dashboard { display: grid; gap: 32px; }
.dashboard-hero { position: relative; min-height: 250px; display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: 36px; padding: clamp(28px, 5vw, 56px); overflow: hidden; color: var(--hero-text); background: var(--hero); border: 1px solid var(--hero-border); border-radius: 14px 14px 14px 4px; }
.dashboard-hero::after { content: ''; position: absolute; inset: 0; opacity: .2; pointer-events: none; background-image: linear-gradient(var(--hero-grid) 1px, transparent 1px), linear-gradient(90deg, var(--hero-grid) 1px, transparent 1px); background-size: 28px 28px; mask-image: linear-gradient(90deg, transparent, #000); }
.dashboard-hero__copy { position: relative; z-index: 1; max-width: 720px; }
.eyebrow { color: var(--accent-strong); font: 700 10px/1 var(--font-display); letter-spacing: .2em; text-transform: uppercase; }
.dashboard-hero h2 { max-width: 700px; margin: 13px 0 14px; font: 650 clamp(30px, 5vw, 58px)/1 var(--font-body); letter-spacing: -.035em; }
.dashboard-hero p { max-width: 620px; margin: 0; color: var(--hero-muted); font-size: 15px; line-height: 1.75; }
.dashboard-hero__actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 28px; }
.dashboard-primary-action, .dashboard-secondary-action, .inline-action { min-height: 44px; display: inline-flex; align-items: center; justify-content: center; gap: 8px; padding: 0 16px; border-radius: 7px; text-decoration: none; font: 650 12px/1 var(--font-display); }
.dashboard-primary-action { color: var(--accent-ink); background: var(--accent); border: 1px solid var(--accent); }
.dashboard-secondary-action { color: var(--hero-text); background: transparent; border: 1px solid var(--hero-border); }
.dashboard-hero__stamp { position: relative; z-index: 1; width: 160px; height: 160px; display: grid; place-content: center; text-align: center; border: 1px solid var(--hero-border); border-radius: 50%; box-shadow: inset 0 0 0 9px color-mix(in srgb, var(--hero) 70%, transparent), inset 0 0 0 10px var(--hero-border); transform: rotate(-4deg); }
.dashboard-hero__stamp span, .dashboard-hero__stamp small { color: var(--hero-muted); font: 600 9px/1.2 var(--font-display); letter-spacing: .12em; text-transform: uppercase; }
.dashboard-hero__stamp strong { margin: 8px 0; color: var(--accent); font: 700 42px/1 var(--font-display); }
.metric-strip { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); background: var(--surface); border: 1px solid var(--border); border-radius: 10px; overflow: hidden; }
.metric-cell { min-height: 126px; display: grid; grid-template-columns: 28px 1fr; gap: 10px; padding: 22px; border-right: 1px solid var(--border); }
.metric-cell:last-child { border-right: 0; }
.metric-cell > span { color: var(--accent-strong); font: 700 9px/1 var(--font-display); }
.metric-cell small, .metric-cell em { display: block; color: var(--text-faint); font-style: normal; }
.metric-cell small { font: 600 10px/1 var(--font-display); letter-spacing: .12em; text-transform: uppercase; }
.metric-cell strong { display: block; margin: 12px 0 8px; color: var(--text); font: 650 28px/1 var(--font-display); }
.metric-cell em { font-size: 12px; }
.dashboard-section { display: grid; gap: 16px; }
.section-heading, .panel-heading { display: flex; justify-content: space-between; align-items: end; gap: 16px; }
.section-heading h3, .panel-heading h3 { margin: 7px 0 0; color: var(--text); font-size: 20px; }
.section-heading a, .panel-heading a { display: inline-flex; align-items: center; gap: 6px; min-height: 44px; color: var(--text-muted); font: 600 12px/1 var(--font-display); text-decoration: none; }
.section-heading a:hover, .panel-heading a:hover { color: var(--accent-strong); }
.platform-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
.platform-card { padding: 20px; background: var(--surface); border: 1px solid var(--border); border-radius: 10px 10px 10px 3px; animation-delay: var(--delay); transition: border-color .2s ease, box-shadow .2s ease; }
.platform-card:hover { border-color: var(--border-strong); box-shadow: 7px 7px 0 var(--shadow-line); }
.platform-card__top, .platform-card__last, .platform-card__actions { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.platform-card__number { color: var(--text-faint); font: 700 10px/1 var(--font-display); }
.platform-card__status { display: inline-flex; align-items: center; gap: 7px; color: var(--success); font: 650 10px/1 var(--font-display); letter-spacing: .08em; text-transform: uppercase; }
.platform-card__status i { width: 7px; height: 7px; border-radius: 50%; background: currentColor; box-shadow: 0 0 0 3px var(--success-dim); }
.platform-card__status.is-off { color: var(--text-faint); }
.platform-card__status.is-off i { box-shadow: none; }
.platform-card__brand { display: flex; align-items: center; gap: 13px; margin: 26px 0 20px; }
.platform-glyph { width: 42px; height: 42px; display: grid; place-items: center; color: var(--text); background: var(--surface-muted); border: 1px solid var(--border); border-radius: 8px; }
.platform-glyph svg { width: 23px; height: 23px; }
.platform-card h4 { margin: 0; color: var(--text); font: 650 17px/1 var(--font-display); }
.platform-card p { margin: 6px 0 0; color: var(--text-faint); font-size: 12px; }
.platform-card__facts { display: grid; grid-template-columns: 1.5fr 1fr .6fr; margin: 0; padding: 14px 0; border-top: 1px solid var(--border); border-bottom: 1px solid var(--border); }
.platform-card__facts div { min-width: 0; padding-right: 9px; }
.platform-card__facts dt { color: var(--text-faint); font: 600 9px/1 var(--font-display); text-transform: uppercase; }
.platform-card__facts dd { margin: 8px 0 0; overflow: hidden; color: var(--text); font: 600 12px/1 var(--font-display); text-overflow: ellipsis; white-space: nowrap; }
.platform-card__last { padding: 15px 0; color: var(--text-faint); font-size: 11px; }
.status-badge { display: inline-flex; align-items: center; min-height: 24px; padding: 0 8px; border-radius: 5px; font: 650 10px/1 var(--font-display); }
.is-success { color: var(--success); background: var(--success-dim); }.is-failed { color: var(--danger); background: var(--danger-dim); }.is-running, .is-cancelling { color: var(--warn); background: var(--warn-dim); }.is-cancelled { color: var(--text-faint); background: var(--surface-muted); }.is-idle { color: var(--text-faint); background: var(--surface-muted); }
.platform-card__actions { justify-content: flex-start; }
.platform-card__actions a { min-height: 44px; display: inline-flex; align-items: center; color: var(--text); font: 650 11px/1 var(--font-display); text-decoration: none; }
.platform-card__actions a:first-child { color: var(--accent-strong); }
.dashboard-lower { display: grid; grid-template-columns: minmax(0, 1.75fr) minmax(280px, .75fr); gap: 14px; }
.ops-panel { padding: 22px; background: var(--surface); border: 1px solid var(--border); border-radius: 10px; }
.activity-list { margin-top: 18px; border-top: 1px solid var(--border); }
.activity-row { position: relative; display: grid; grid-template-columns: 64px minmax(150px, 1fr) 150px auto; align-items: center; gap: 14px; min-height: 62px; padding-left: 12px; border-bottom: 1px solid var(--border); }
.activity-row__line { position: absolute; left: 0; top: 13px; bottom: 13px; width: 3px; padding: 0; }
.activity-row__id { color: var(--text-faint); font: 600 10px/1 var(--font-display); }
.activity-row__main { min-width: 0; }
.activity-row__main strong, .activity-row__main span { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.activity-row__main strong { color: var(--text); font: 650 12px/1 var(--font-display); }
.activity-row__main span, .activity-row time { margin-top: 5px; color: var(--text-faint); font-size: 11px; }
.status-panel__rate { margin: 28px 0 18px; }
.status-panel__rate strong { display: block; color: var(--text); font: 700 52px/1 var(--font-display); letter-spacing: -.05em; }
.status-panel__rate span { color: var(--text-faint); font-size: 12px; }
.quality-bar { height: 10px; display: flex; overflow: hidden; background: var(--surface-muted); border-radius: 3px; }
.quality-bar i { display: block; min-width: 0; border-radius: 0; }
.quality-list { display: grid; gap: 12px; margin: 22px 0; }
.quality-list div { display: flex; justify-content: space-between; }
.quality-list dt { display: flex; align-items: center; gap: 8px; color: var(--text-muted); font-size: 12px; }
.quality-list dt i { width: 8px; height: 8px; display: inline-block; padding: 0; border-radius: 2px; }
.quality-list dd { color: var(--text); font: 650 12px/1 var(--font-display); }
.status-panel__note { display: flex; gap: 9px; padding-top: 17px; color: var(--text-faint); border-top: 1px solid var(--border); font-size: 11px; line-height: 1.6; }
.inline-action { color: var(--accent-ink); background: var(--accent); }
@media (max-width: 1180px) { .platform-grid { grid-template-columns: 1fr; }.dashboard-lower { grid-template-columns: 1fr; } }
@media (max-width: 860px) { .metric-strip { grid-template-columns: 1fr 1fr; }.metric-cell:nth-child(2) { border-right: 0; }.metric-cell:nth-child(-n+2) { border-bottom: 1px solid var(--border); }.dashboard-hero__stamp { width: 130px; height: 130px; } }
@media (max-width: 640px) { .dashboard-hero { grid-template-columns: 1fr; }.dashboard-hero__stamp { display: none; }.dashboard-hero h2 { font-size: 34px; }.metric-strip { grid-template-columns: 1fr; }.metric-cell { border-right: 0; border-bottom: 1px solid var(--border); }.metric-cell:last-child { border-bottom: 0; }.activity-row { grid-template-columns: 54px 1fr auto; }.activity-row time { display: none; }.section-heading { align-items: flex-start; }.platform-card__facts { grid-template-columns: 1fr 1fr; }.platform-card__facts div:last-child { display: none; } }
</style>
