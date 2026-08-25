<template>
  <el-config-provider :locale="epLocale">
    <router-view />
  </el-config-provider>
</template>

<script setup>
import { computed } from 'vue'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import zhTw from 'element-plus/es/locale/lang/zh-tw'
import en from 'element-plus/es/locale/lang/en'
import { currentLocale } from './i18n'

// 让 Element Plus 内置文案（分页、空态、时间选择器等）随当前语言切换
const epMap = { 'zh-CN': zhCn, 'zh-TW': zhTw, 'en': en }
const epLocale = computed(() => epMap[currentLocale()] || zhCn)
</script>

<style>
html, body, #app { min-height: 100%; margin: 0; }
</style>
