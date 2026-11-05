<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

type Theme = 'system' | 'light' | 'dark'
const modes: Theme[] = ['system', 'light', 'dark']
const labels = { system: '跟随系统', light: '浅色', dark: '深色' }
const theme = ref<Theme>((document.documentElement.dataset.themePreference as Theme) || 'system')
const hintDismissed = ref(false)
const system = window.matchMedia('(prefers-color-scheme: dark)')
const next = computed(() => modes[(modes.indexOf(theme.value) + 1) % modes.length]!)
const hint = computed(() => `当前：${labels[theme.value]}；点击切换为${labels[next.value]}`)

function applyTheme() {
  const root = document.documentElement
  root.dataset.theme = theme.value === 'system' ? (system.matches ? 'dark' : 'light') : theme.value
  root.style.colorScheme = root.dataset.theme
  root.dataset.themePreference = theme.value
  document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')!.content = getComputedStyle(root).getPropertyValue('--page').trim()
}

function cycleTheme() {
  theme.value = next.value
  applyTheme()
  try {
    localStorage.setItem('wordweave.theme', theme.value)
  } catch {
    // Switching appearance still works without browser storage.
  }
}

onMounted(() => {
  applyTheme()
  system.addEventListener('change', applyTheme)
})
onBeforeUnmount(() => system.removeEventListener('change', applyTheme))
</script>

<template>
  <div class="theme-control" :class="{ 'hint-dismissed': hintDismissed }" @pointerenter="hintDismissed = false" @keydown.esc="hintDismissed = true">
    <button type="button" class="theme-switch" :aria-label="`外观：当前${labels[theme]}；点击切换为${labels[next]}`" aria-describedby="theme-hint" @focus="hintDismissed = false" @click="cycleTheme">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        <template v-if="theme === 'system'">
          <rect x="3" y="4" width="18" height="13" rx="2" />
          <path d="M8 21h8m-4-4v4" />
        </template>
        <template v-else-if="theme === 'light'">
          <circle cx="12" cy="12" r="4" />
          <path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5" />
        </template>
        <path v-else d="M20.5 14A8.5 8.5 0 0 1 10 3.5 8.5 8.5 0 1 0 20.5 14Z" />
      </svg>
    </button>
    <span id="theme-hint" role="tooltip" class="theme-hint">{{ hint }}</span>
  </div>
</template>
