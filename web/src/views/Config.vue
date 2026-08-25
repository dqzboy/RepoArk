<template>
  <div v-loading="loading" element-loading-background="transparent">
    <PlatformSwitcher v-model="active" :enabled-map="enabledMap" @change="onTabChange" />

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
          <div style="width: 100%">
            <el-input
              v-model="form.git_token"
              :type="tokenVisible ? 'text' : 'password'"
              :placeholder="t('config.tokenPh')"
              @focus="prepareTokenEdit"
              @input="markTokenDirty"
            >
              <template #suffix>
                <el-icon
                  class="token-visibility"
                  :title="tokenVisible ? t('config.hideToken') : t('config.showToken')"
                  @mousedown.prevent.stop
                  @click.stop="toggleTokenVisibility"
                >
                  <Hide v-if="tokenVisible" />
                  <View v-else />
                </el-icon>
              </template>
            </el-input>
            <div class="field-help">
              {{ tokenConfigured ? t('config.tokenConfigured') : t('config.tokenNotConfigured') }}
            </div>
          </div>
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
          <div class="field-help">{{ t('config.backupDirHelp') }}</div>
        </el-form-item>
        <el-form-item :label="t('config.serverName')" required>
          <el-input v-model="form.server_name" :placeholder="t('config.serverNamePh')" />
          <div class="field-help">{{ t('config.serverNameHelp') }}</div>
        </el-form-item>
        <el-form-item :label="t('config.hostRoot')">
          <el-input v-model="form.host_root" :placeholder="t('config.hostRootPh')" />
          <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 6px; line-height: 1.5">{{ t('config.hostRootHelp') }}</div>
        </el-form-item>
        <el-form-item :label="t('config.sources')" required>
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
import { Plus, Delete, Setting, View, Hide } from '@element-plus/icons-vue'
import PlatformSwitcher from '../components/PlatformSwitcher.vue'
import api from '../api'
import { t } from '../i18n'
import { usePlatformStore, platformLabel as pLabel } from '../stores/platform'

const store = usePlatformStore()
const active = ref(store.platform)
const platform = computed(() => pLabel(active.value))
const TOKEN_MASK = '********'
const tokenConfigured = ref(false)
const tokenVisible = ref(false)
const tokenDirty = ref(false)
const tokenRevealed = ref(false)

const form = ref({
  enabled: false,
  git_user: '',
  git_token: '',
  repo_name: '',
  branch: '',
  backup_dir: '',
  server_name: '',
  host_root: '',
  backup_sources: [],
  schedule_enabled: false,
  schedule_cron: ''
})
const enabledMap = ref({})
const saving = ref(false)
const loading = ref(true)

async function loadPlatform(code) {
  try {
    const { data } = await api.get('/platforms/' + code)
    tokenConfigured.value = !!data.token_configured || data.git_token === TOKEN_MASK
    tokenVisible.value = false
    tokenDirty.value = false
    tokenRevealed.value = false
    form.value = {
      enabled: !!data.enabled,
      git_user: data.git_user || '',
      git_token: tokenConfigured.value ? TOKEN_MASK : '',
      repo_name: data.repo_name || '',
      branch: data.branch || '',
      backup_dir: data.backup_dir || '',
      server_name: data.server_name || '',
      host_root: data.host_root || '',
      backup_sources: Array.isArray(data.backup_sources) ? [...data.backup_sources] : [],
      schedule_enabled: !!data.schedule_enabled,
      schedule_cron: data.schedule_cron || ''
    }
    enabledMap.value[code] = !!data.enabled
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('config.saveError'))
  }
}

function prepareTokenEdit() {
  if (!tokenVisible.value && !tokenDirty.value && form.value.git_token === TOKEN_MASK) {
    form.value.git_token = ''
  }
}

function markTokenDirty() {
  tokenDirty.value = true
}

async function toggleTokenVisibility() {
  if (tokenVisible.value) {
    tokenVisible.value = false
    if (tokenRevealed.value && !tokenDirty.value) {
      form.value.git_token = TOKEN_MASK
      tokenRevealed.value = false
    }
    return
  }

  if (tokenConfigured.value && !tokenDirty.value) {
    try {
      const { data } = await api.get('/platforms/' + active.value + '/token')
      form.value.git_token = data.git_token || ''
      tokenRevealed.value = true
    } catch (e) {
      ElMessage.error(e.response?.data?.error || t('config.tokenLoadError'))
      return
    }
  }
  tokenVisible.value = true
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
  const nodeName = (form.value.server_name || '').trim()
  if (!nodeName) {
    ElMessage.warning(t('config.nodeRequired'))
    return
  }
  const sources = (form.value.backup_sources || []).map((s) => s.trim()).filter(Boolean)
  if (sources.length === 0) {
    ElMessage.warning(t('config.sourceRequired'))
    return
  }
  saving.value = true
  try {
    const payload = {
      enabled: !!form.value.enabled,
      git_user: form.value.git_user,
      git_token: tokenDirty.value ? form.value.git_token : '',
      repo_name: form.value.repo_name,
      branch: form.value.branch,
      backup_dir: form.value.backup_dir,
      server_name: nodeName,
      host_root: form.value.host_root,
      backup_sources: sources,
      schedule_enabled: !!form.value.schedule_enabled,
      schedule_cron: form.value.schedule_cron || ''
    }
    await api.put('/platforms/' + active.value, payload)
    enabledMap.value[active.value] = !!form.value.enabled
    if (tokenDirty.value && form.value.git_token) tokenConfigured.value = true
    form.value.git_token = tokenConfigured.value ? TOKEN_MASK : ''
    tokenVisible.value = false
    tokenDirty.value = false
    tokenRevealed.value = false
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
.field-help {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.token-visibility {
  cursor: pointer;
}
</style>
