<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage, formatDate, getEntry, type Entry } from '../api'

const props = defineProps<{ id: string }>()
const entry = ref<Entry>()
const error = ref('')
const loading = ref(true)

async function load() {
  loading.value = true
  error.value = ''
  try {
    entry.value = await getEntry(props.id)
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    loading.value = false
  }
}
onMounted(load)
</script>

<template>
  <RouterLink class="back-link" to="/">← 返回全部词条</RouterLink>
  <div v-if="error" class="notice error" role="alert">
    {{ error }} <button class="text-button" @click="load">重试</button>
  </div>
  <p v-else-if="loading" role="status" class="loading">正在读取词条…</p>
  <template v-else-if="entry">
    <div class="page-heading">
      <div>
        <p class="eyebrow">IN YOUR COLLECTION</p>
        <h1 class="word-title" lang="en">{{ entry.spelling }}</h1>
        <p class="subtitle">
          收录于
          <time :datetime="entry.createdAt">{{
            formatDate(entry.createdAt)
          }}</time>
        </p>
      </div>
      <RouterLink class="button secondary" to="/entries/new"
        >＋ 继续收词</RouterLink
      >
    </div>
    <section class="detail-card">
      <h2>释义<span class="section-line"></span></h2>
      <p class="definition" :class="{ muted: !entry.definition }">
        {{ entry.definition || '暂未填写释义' }}
      </p>
    </section>
    <section class="detail-card">
      <h2>
        关系组<span class="small-count">0</span
        ><span class="section-line"></span>
      </h2>
      <div class="group-empty">
        <span aria-hidden="true">○ — ○</span>
        <p>暂无关系组</p>
        <small>这个词条还没有关联到其他词条。</small>
      </div>
    </section>
  </template>
</template>
