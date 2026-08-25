// 跨页面共享的「当前选中平台」状态：用在 Layout 顶部 tabs、Config / Schedule / Backup 三处。
// 持久化到 localStorage，刷新后保留用户上次选择；默认 github。
import { reactive, watch } from 'vue'

const STORAGE_KEY = 'ops-active-platform'
const DEFAULT = 'github'

export const PLATFORMS = [
  { code: 'github',  icon: 'M12 .5C5.65.5.5 5.65.5 12c0 5.07 3.29 9.36 7.86 10.88.58.1.79-.25.79-.56 0-.28-.01-1.02-.02-2-3.2.7-3.87-1.54-3.87-1.54-.52-1.34-1.28-1.7-1.28-1.7-1.04-.71.08-.7.08-.7 1.15.08 1.76 1.18 1.76 1.18 1.03 1.76 2.7 1.25 3.36.95.1-.74.4-1.25.73-1.54-2.55-.29-5.24-1.28-5.24-5.7 0-1.26.45-2.3 1.18-3.1-.12-.29-.51-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11.04 11.04 0 0 1 5.8 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.24 2.76.12 3.05.74.8 1.18 1.84 1.18 3.1 0 4.43-2.69 5.41-5.26 5.7.41.36.78 1.06.78 2.13 0 1.54-.01 2.78-.01 3.16 0 .31.21.67.79.56C20.21 21.36 23.5 17.07 23.5 12 23.5 5.65 18.35.5 12 .5z' },
  { code: 'gitcode', icon: 'M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20zm-1 5h2v6h-2V7zm0 8h2v2h-2v-2z' },
  { code: 'gitee',   icon: 'M3 5h18v2H3V5zm0 4h18v2H3V9zm0 4h12v2H3v-2zm0 4h8v2H3v-2z' }
]

const state = reactive({
  platform: readStored() || DEFAULT
})

function readStored() {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v && PLATFORMS.some((p) => p.code === v)) return v
  } catch (_) { /* localStorage unavailable */ }
  return null
}

watch(() => state.platform, (v) => {
  try { localStorage.setItem(STORAGE_KEY, v) } catch (_) {}
}, { immediate: true })

export function usePlatformStore() {
  return state
}

export function platformLabel(code) {
  switch (code) {
    case 'github': return 'GitHub'
    case 'gitcode': return 'GitCode'
    case 'gitee': return 'Gitee'
    default: return code
  }
}
