<script setup lang="ts">
import { computed, nextTick, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { isSaving } from './leaveProtection'
import { getStats, notebookRevision } from './api'

const route = useRoute()
const router = useRouter()
const query = ref('')
const saveNotice = ref(false)
function blocked() { saveNotice.value = true }
function updateSaveNotice() { if (!isSaving()) saveNotice.value = false }
onMounted(() => { window.addEventListener('save-navigation-blocked', blocked); window.addEventListener('save-state-changed', updateSaveNotice) })
onBeforeUnmount(() => { window.removeEventListener('save-navigation-blocked', blocked); window.removeEventListener('save-state-changed', updateSaveNotice) })
watch(() => route.fullPath, async () => {
  saveNotice.value = false
  document.title = (route.path === '/' ? '全部词条' : route.path === '/relations' ? '关系浏览' : route.path.includes('/new') ? '收录与建组' : route.path.startsWith('/groups/') ? '编辑关系组' : '词条详情') + ' · Wordweave'
  await nextTick()
  document.getElementById('main')?.focus({ preventScroll: true })
}, { immediate: true })
const browsing = computed(() => route.path === '/' || route.path === '/relations')
const totals = ref<{ entries: number; groups: number }>()
const statsError = ref(false)
let statsRequest = 0

async function refreshStats() {
  const current = ++statsRequest
  statsError.value = false
  try {
    const result = await getStats()
    if (current === statsRequest) totals.value = result
  } catch {
    if (current === statsRequest) {
      totals.value = undefined
      statsError.value = true
    }
  }
}
watch([() => route.fullPath, notebookRevision], refreshStats, { immediate: true })

watch(() => route.fullPath, () => {
  query.value = route.path === '/' && typeof route.query.q === 'string' ? route.query.q : ''
}, { immediate: true })

function submitSearch() {
  const q = query.value.trim()
  void router.push({ path: '/', query: q ? { q } : {} })
}
</script>

<template>
  <a class="skip-link" href="#main">跳至主要内容</a>
  <div class="layout">
    <aside class="sidebar">
      <RouterLink class="brand" to="/" aria-label="Wordweave 生词本首页">
        <span class="brand-mark" aria-hidden="true">W<span>·</span></span>
        <span>Wordweave<small>我的生词本</small></span>
      </RouterLink>
      <div class="nav-label">我的学习空间</div>
      <nav aria-label="主要导航">
        <RouterLink class="nav-link" to="/" exact-active-class="selected">
          <span aria-hidden="true">▤</span> 全部词条
        </RouterLink>
        <RouterLink class="nav-link" to="/relations" active-class="selected">
          <span aria-hidden="true">◇</span> 关系浏览
        </RouterLink>
      </nav>
      <div class="sidebar-note">
        <span class="status-dot"></span> 本机生词本<small
          >一点一滴，织成自己的词汇。</small
        >
      </div>
    </aside>
    <div class="workspace">
      <header class="topbar">
        <form class="global-search" role="search" @submit.prevent="submitSearch">
          <label for="global-search" class="sr-only">搜索词条与关系说明</label>
          <span class="search-icon" aria-hidden="true">⌕</span>
          <input id="global-search" v-model="query" name="q" type="search" placeholder="搜索拼写、释义或关系说明" />
          <button type="submit">搜索 <span aria-hidden="true">↵</span></button>
        </form>
        <div class="notebook-totals" aria-live="polite" aria-atomic="true">
          <span v-if="totals">词条 <strong>{{ totals.entries }}</strong> · 关系组 <strong>{{ totals.groups }}</strong></span>
          <button v-else-if="statsError" type="button" class="text-button" @click="refreshStats">统计暂不可用，重试</button>
          <span v-else>正在统计…</span>
        </div>
      </header>
      <p v-if="saveNotice" role="status" class="notice">正在保存，请稍候</p>
      <main id="main" tabindex="-1" :class="{ 'browse-main': browsing }"><RouterView :key="$route.fullPath" /></main>
      <footer>WORDWEAVE <span>让遇见的词，成为自己的语言。</span></footer>
    </div>
  </div>
</template>
