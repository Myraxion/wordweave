<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage, formatDate, listEntries, type Entry } from '../api'

const entries = ref<Entry[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    entries.value = await listEntries()
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    loading.value = false
  }
}
onMounted(load)
</script>

<template>
  <div class="page-heading">
    <div>
      <p class="eyebrow">YOUR WORD COLLECTION</p>
      <h1>
        全部词条<span class="count">{{ entries.length }}</span>
      </h1>
      <p class="subtitle">把遇见的新词留在这里，慢慢读懂，慢慢积累。</p>
    </div>
    <RouterLink class="button primary" to="/entries/new"
      ><span aria-hidden="true">＋</span> 收录词条</RouterLink
    >
  </div>
  <div v-if="error" class="notice error" role="alert">
    {{ error }} <button class="text-button" @click="load">重试</button>
  </div>
  <p v-else-if="loading" role="status" class="loading">正在翻开生词本…</p>
  <section v-else-if="entries.length === 0" class="empty-card">
    <div class="empty-symbol" aria-hidden="true">Aa<span>＋</span></div>
    <h2>从第一个新词开始</h2>
    <p>一个单词，或一句短语。<br />先记下来，释义可以慢慢补充。</p>
    <RouterLink class="button primary" to="/entries/new"
      >收录第一个词条 <span aria-hidden="true">→</span></RouterLink
    >
  </section>
  <section v-else class="entry-card" aria-label="词条列表">
    <div class="list-heading">
      <h2>
        已收录 <span>{{ entries.length }} 个词条</span>
      </h2>
      <span>最近添加在前</span>
    </div>
    <ul class="entry-list">
      <li v-for="entry in entries" :key="entry.id">
        <RouterLink :to="`/entries/${entry.id}`" class="entry-row">
          <div class="entry-copy">
            <h3 lang="en">{{ entry.spelling }}</h3>
            <p :class="{ muted: !entry.definition }">
              {{ entry.definition || '暂未填写释义' }}
            </p>
          </div>
          <time :datetime="entry.createdAt">{{
            formatDate(entry.createdAt)
          }}</time
          ><span class="row-arrow" aria-hidden="true">↗</span>
        </RouterLink>
      </li>
    </ul>
  </section>
</template>
