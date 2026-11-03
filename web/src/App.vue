<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const query = ref('')
const browsing = computed(() => route.path === '/' || route.path === '/relations')

watch(() => route.fullPath, () => {
  query.value = route.path === '/' && typeof route.query.q === 'string' ? route.query.q : ''
}, { immediate: true })

function submitSearch() {
  const q = query.value.trim()
  void router.push({ path: '/', query: q ? { q } : {} })
}
</script>

<template>
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
          <span aria-hidden="true">◇</span> 关系类型浏览
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
        <span class="local-badge">个人词汇收藏</span>
      </header>
      <main id="main" :class="{ 'browse-main': browsing }"><RouterView :key="$route.fullPath" /></main>
      <footer>WORDWEAVE <span>让遇见的词，成为自己的语言。</span></footer>
    </div>
  </div>
</template>
