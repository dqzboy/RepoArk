<template>
  <div class="platform-switcher" role="tablist" :aria-label="t('platform.tab')">
    <button
      v-for="item in PLATFORMS"
      :key="item.code"
      type="button"
      role="tab"
      class="platform-switcher__item"
      :class="{ 'is-active': modelValue === item.code }"
      :aria-selected="modelValue === item.code"
      @click="$emit('update:modelValue', item.code); $emit('change', item.code)"
    >
      <span class="platform-switcher__index">{{ platformIndex(item.code) }}</span>
      <span class="platform-switcher__name">{{ t('platform.' + item.code) }}</span>
      <span class="platform-switcher__state" :class="{ 'is-off': !enabledMap[item.code] }">
        <span class="platform-switcher__dot" />
        {{ enabledMap[item.code] ? t('platform.ready') : t('platform.offline') }}
      </span>
    </button>
  </div>
</template>

<script setup>
import { t } from '../i18n'
import { PLATFORMS } from '../stores/platform'

defineProps({
  modelValue: { type: String, required: true },
  enabledMap: { type: Object, default: () => ({}) }
})
defineEmits(['update:modelValue', 'change'])

function platformIndex(code) {
  return { github: '01', gitcode: '02', gitee: '03' }[code] || '--'
}
</script>

<style scoped>
.platform-switcher {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
  margin-bottom: 20px;
  padding: 6px;
  background: var(--surface-muted);
  border: 1px solid var(--border);
  border-radius: 12px;
}
.platform-switcher__item {
  min-height: 54px;
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: var(--text-muted);
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: color .2s ease, background-color .2s ease, border-color .2s ease;
}
.platform-switcher__item:hover { color: var(--text); background: var(--surface); }
.platform-switcher__item.is-active {
  color: var(--text);
  background: var(--surface);
  border-color: var(--border-strong);
  box-shadow: 0 2px 0 var(--shadow-line);
}
.platform-switcher__item:focus-visible { outline: 3px solid var(--focus); outline-offset: 2px; }
.platform-switcher__index { font: 600 10px/1 var(--font-display); color: var(--accent-strong); letter-spacing: .12em; }
.platform-switcher__name { font: 600 13px/1.2 var(--font-display); }
.platform-switcher__state { display: inline-flex; align-items: center; gap: 6px; font-size: 11px; }
.platform-switcher__dot { width: 7px; height: 7px; border-radius: 50%; background: var(--success); box-shadow: 0 0 0 3px var(--success-dim); }
.platform-switcher__state.is-off { color: var(--text-faint); }
.platform-switcher__state.is-off .platform-switcher__dot { background: var(--text-faint); box-shadow: none; }
@media (max-width: 720px) {
  .platform-switcher { grid-template-columns: 1fr; }
  .platform-switcher__item { min-height: 48px; }
}
</style>
