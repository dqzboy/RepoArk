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
        <span class="ops-card__title">
          <el-icon class="ops-card__icon"><Setting /></el-icon>{{ t('config.title') }}
          <el-tag v-if="!form.enabled" type="info" size="small" style="margin-left: 8px">{{ t('common.none') }}</el-tag>
        </span>
      </template>

      <el-alert
        v-if="!form.enabled"
        type="warning"
        :closable="false"
        :title="t('platform.disabledHint')"
        style="margin-bottom: 16px"
      />

      <el-form :model="form" label-width="160px" style="max-width: 760px">
        <el-form-item :label="t('common.enablePlatform')">
          <el-switch v-model="form.enabled" />
        </el-form-item>

        <div class="ops-section"><span>{{ t('config.sectionRepo', { platform }) }}</span></div>
        <el-form-item :label="t('config.gitUser', { platform })" required>
          <el-input v-model="form.git_user" />
        </el-form-item>
        <el-form-item :label="t('config.gitToken', { platform })" required>
          <el-input v-model="form.git_token" type="password" show-password :placeholder="t('config.tokenPh')" />
        </el-form-item>
        <el-form-item :label="t('config.repoName')" required>
          <el-input v-model="form.repo_name" />
        </el-form-item>
        <el-form-item :label="t('config.branch')">
          <el-input v-model="form.branch" />
        </el-form-item>

        <div class="ops-section"><span>{{ t('config.sectionBackup') }}</span></div>
        <el-form-item :label="t('config.backupDir')" required>
          <el-input v-model="form.backup_dir" />
        </el-form-item>
        <el-form-item :label="t('config.serverName')">
          <el-input v-model="form.server_name" :placeholder="t('config.serverNamePh')" />
        </el-form-item>
        <el-form-item :label="t('config.hostRoot')">
          <el-input v-model="form.host_root" :placeholder="t('config.hostRootPh')" />
          <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 6px; line-height: 1.5">{{ t('config.hostRootHelp') }}</div>
        </el-form-item>
        <el-form-item :label="t('config.sources')">
          <div style="width: 100%">
            <div v-for="(src, i) in form.backup_sources" :key="i" style="display: flex; margin-bottom: 10px">
              <el-input v-model="form.backup_sources[i]" :placeholder="t('config.sourcePh')" />
              <el-button type="danger" :icon="Delete" circle style="margin-left: 10px" @click="removeSource(i)" />
            </div>
            <el-button :icon="Plus" @click="addSource">{{ t('config.addPath') }}</el-button>
          </div>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="saving" @click="save">{{ t('config.save') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Delete, Setting } from '@element-plus/icons-vue'
import api from '../api'
import { t } from '../i18n'
import { PLATFORMS, usePlatformStore, platformLabel as pLabel } from '../stores/platform'

const store = usePlatformStore()
const active = ref(store.platform)
const platform = computed(() => pLabel(active.value))

const form = ref({
  enabled: false,
  git_user: '',
  git_token: '',
  repo_name: '',
  branch: '',
  backup_dir: '',
  server_name: '',
  host_root: '',
  backup_sources: []
})
const enabledMap = ref({})
const saving = ref(false)
const loading = ref(true)

function dotColor(code) {
  if (code === 'github') return '#7c5cff'
  if (code === 'gitcode') return '#22b8ff'
  return '#ff6b35'
}

async function loadPlatform(code) {
  try {
    const { data } = await api.get('/platforms/' + code)
    form.value = {
      enabled: !!data.enabled,
      git_user: data.git_user || '',
      git_token: data.git_token || '',
      repo_name: data.repo_name || '',
      branch: data.branch || '',
      backup_dir: data.backup_dir || '',
      server_name: data.server_name || '',
      host_root: data.host_root || '',
      backup_sources: Array.isArray(data.backup_sources) ? [...data.backup_sources] : []
    }
    enabledMap.value[code] = !!data.enabled
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('config.saveError'))
  }
}

async function loadAllEnabledMap() {
  try {
    const { data } = await api.get('/platforms')
    const m = {}
    for (const p of data || []) m[p.platform] = !!p.enabled
    enabledMap.value = m
  } catch (e) { /* ignore */ }
}

async function onTabChange(tab) {
  store.platform = tab
  await loadPlatform(tab)
}

function addSource() {
  form.value.backup_sources.push('')
}
function removeSource(i) {
  form.value.backup_sources.splice(i, 1)
}

async function save() {
  saving.value = true
  try {
    const payload = {
      enabled: !!form.value.enabled,
      git_user: form.value.git_user,
      git_token: form.value.git_token,
      repo_name: form.value.repo_name,
      branch: form.value.branch,
      backup_dir: form.value.backup_dir,
      server_name: form.value.server_name,
      host_root: form.value.host_root,
      backup_sources: (form.value.backup_sources || []).map((s) => s.trim()).filter(Boolean),
      schedule_enabled: false, // schedule_enabled cron 在 /Schedule.vue 中维护
      schedule_cron: ''
    }
    await api.put('/platforms/' + active.value, payload)
    enabledMap.value[active.value] = !!form.value.enabled
    ElMessage.success(t('config.saved'))
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('config.saveError'))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  await loadAllEnabledMap()
  await loadPlatform(active.value)
  loading.value = false
})

watch(() => store.platform, async (v) => {
  if (v && v !== active.value) {
    active.value = v
    await loadPlatform(v)
  }
})
</script>

<style scoped>
.ops-platform-tabs {
  margin-bottom: 14px;
}
.ops-platform-tabs :deep(.el-tabs__nav-wrap::after) {
  background-color: transparent;
}
.ops-platform-tab {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
}
.ops-platform-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  box-shadow: 0 0 6px currentColor;
  transition: opacity 0.2s ease;
}
.ops-platform-dot--off {
  opacity: 0.35;
  box-shadow: none;
}
.ops-section {
  display: flex;
  align-items: center;
  margin: 6px 0 14px;
  font-family: var(--font-display);
  font-size: 11px;
  letter-spacing: 0.18em;
  color: var(--text-muted);
  text-transform: uppercase;
}
.ops-section span {
  background: var(--surface-2);
  border: 1px solid var(--border);
  padding: 4px 10px;
  border-radius: 6px;
}
</style>
