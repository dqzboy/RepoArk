<template>
  <el-card class="ops-card">
    <template #header>
      <div class="ops-users-head">
        <span class="ops-card__title"><el-icon class="ops-card__icon"><User /></el-icon>{{ t('users.title') }}</span>
        <el-button type="primary" :icon="Plus" @click="openCreate">{{ t('users.add') }}</el-button>
      </div>
    </template>

    <el-table :data="users" style="width: 100%" v-loading="loading" element-loading-background="transparent">
      <el-table-column prop="id" :label="t('users.id')" width="80" />
      <el-table-column prop="username" :label="t('users.username')" />
      <el-table-column :label="t('users.role')" width="150">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'success' : 'info'" effect="dark">
            {{ row.role === 'admin' ? t('users.admin') : t('users.viewer') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" :label="t('users.created')" width="200" />
      <el-table-column :label="t('users.actions')" width="160">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">{{ t('users.edit') }}</el-button>
          <el-button link type="danger" @click="remove(row)">{{ t('users.delete') }}</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <EmptyState :title="t('users.empty')" />
      </template>
    </el-table>

    <el-dialog
      v-model="dialog"
      :title="editing ? t('users.editTitle') : t('users.createTitle')"
      width="460px"
      append-to-body
      align-center
      destroy-on-close
    >
      <el-form :model="form" label-width="80px">
        <el-form-item :label="t('users.username')" required>
          <el-input v-model="form.username" :disabled="editing" :placeholder="t('users.usernamePh')" />
        </el-form-item>
        <el-form-item :label="editing ? t('users.resetPwd') : t('users.password')" :required="!editing">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :placeholder="editing ? t('users.pwdResetPh') : t('users.pwdPh')"
          />
        </el-form-item>
        <el-form-item :label="t('users.roleL')">
          <el-select v-model="form.role" style="width: 100%">
            <el-option :label="t('users.roleAdmin')" value="admin" />
            <el-option :label="t('users.roleViewer')" value="viewer" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">{{ t('users.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="submit">{{ t('users.save') }}</el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, User } from '@element-plus/icons-vue'
import EmptyState from '../components/EmptyState.vue'
import api from '../api'
import { t } from '../i18n'

const users = ref([])
const dialog = ref(false)
const editing = ref(false)
const saving = ref(false)
const loading = ref(true)
const form = ref({ id: 0, username: '', password: '', role: 'viewer' })

async function load() {
  loading.value = true
  try {
    const { data } = await api.get('/users')
    users.value = data
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = false
  form.value = { id: 0, username: '', password: '', role: 'viewer' }
  dialog.value = true
}
function openEdit(row) {
  editing.value = true
  form.value = { id: row.id, username: row.username, password: '', role: row.role }
  dialog.value = true
}

async function submit() {
  saving.value = true
  try {
    if (editing.value) {
      const payload = { role: form.value.role }
      if (form.value.password) payload.password = form.value.password
      await api.put('/users/' + form.value.id, payload)
    } else {
      await api.post('/users', {
        username: form.value.username,
        password: form.value.password,
        role: form.value.role
      })
    }
    ElMessage.success(t('users.saved'))
    dialog.value = false
    await load()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('users.saveError'))
  } finally {
    saving.value = false
  }
}

async function remove(row) {
  try {
    await ElMessageBox.confirm(t('users.delConfirm', { user: row.username }), t('users.delTitle'), { type: 'warning' })
  } catch (e) {
    return
  }
  try {
    await api.delete('/users/' + row.id)
    ElMessage.success(t('users.deleted'))
    await load()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || t('users.delError'))
  }
}

onMounted(load)
</script>

<style scoped>
.ops-users-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
@media (max-width: 520px) {
  .ops-users-head .el-button {
    width: 100%;
  }
}
</style>
