<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { errorMessage, formatDate, formatDateTime, search, type Entry, type Group } from '../api'
import DisplayModeSwitch from '../components/DisplayModeSwitch.vue'
import GroupCard from '../components/GroupCard.vue'
import BrowseGrid from '../components/BrowseGrid.vue'
import LayoutSwitch from '../components/LayoutSwitch.vue'
import StudyText from '../components/StudyText.vue'
import { useBrowseLayout } from '../browseLayout'
import { studyMode } from '../studyMode'

const route = useRoute()
const query = computed(() => typeof route.query.q === 'string' ? route.query.q.trim() : '')
const sortOrder = ref('updated_desc')
const layout = useBrowseLayout('wordweave.entry-layout')
let requestNumber = 0
const entries = ref<Entry[]>([])
const groups = ref<Group[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  const currentRequest = ++requestNumber
  loading.value = true
  error.value = ''
  try {
    const result = await search(sortOrder.value, query.value)
    if (currentRequest === requestNumber) {
      entries.value = result.entries
      groups.value = result.groups
    }
  } catch (err) {
    if (currentRequest === requestNumber) error.value = errorMessage(err)
  } finally {
    if (currentRequest === requestNumber) loading.value = false
  }
}
onMounted(load)
</script>

<template>
  <div class="browse-page">
    <div class="page-heading">
      <div>
        <p class="eyebrow">YOUR WORD COLLECTION</p>
        <h1>{{ query ? '搜索结果' : '全部词条' }}</h1>
        <p v-if="query" class="subtitle search-summary">搜索“{{ query }}” <RouterLink class="inline-link" to="/">清除搜索</RouterLink></p>
        <p v-else class="subtitle">收集词条，也留一点时间温习。</p>
      </div>
      <RouterLink class="button primary" to="/entries/new"><span aria-hidden="true">＋</span> 收录词条</RouterLink>
    </div>

    <div class="browse-toolbar">
      <DisplayModeSwitch />
      <div class="browse-options">
        <LayoutSwitch v-model="layout" label="词条布局" />
        <label for="sort-select" class="sr-only">排序</label>
        <select id="sort-select" v-model="sortOrder" class="sort-select" @change="load">
          <option value="updated_desc">最近修改在前</option>
          <option value="updated_asc">最早修改在前</option>
          <option value="created_desc">最近添加在前</option>
          <option value="created_asc">最早添加在前</option>
          <option value="alpha_asc">英文 A–Z</option>
          <option value="alpha_desc">英文 Z–A</option>
        </select>
      </div>
    </div>

    <div v-if="error" class="notice error" role="alert">
      {{ error }} <button class="text-button" @click="load">重试</button>
    </div>
    <p v-else-if="loading" role="status" class="loading">正在翻开生词本…</p>
    <template v-else>
      <section v-if="entries.length === 0 && !query" class="empty-card">
        <div class="empty-symbol" aria-hidden="true">Aa<span>＋</span></div>
        <h2>从第一个新词开始</h2>
        <p>一个单词，或一句短语。<br />先记下来，释义可以慢慢补充。</p>
        <RouterLink class="button primary" to="/entries/new">收录第一个词条 <span aria-hidden="true">→</span></RouterLink>
      </section>
      <section v-else aria-label="词条列表">
        <div class="results-heading">
          <h2>{{ query ? '直接命中的词条' : '已收录' }} <span class="small-count">{{ entries.length }} 个词条</span></h2>
        </div>
        <BrowseGrid as="ul" class="entry-list compact-entry-list" :items="entries" :layout="layout">
          <li v-for="entry in entries" :key="entry.id" class="compact-entry-row">
            <div class="entry-copy">
              <h3 lang="en" :class="{ 'masked-entry-heading': studyMode === 'hide-spelling' }">
                <StudyText v-if="studyMode === 'hide-spelling'" :text="entry.spelling" masked label="拼写" kind="spelling" />
                <RouterLink v-else :to="`/entries/${entry.id}`">{{ entry.spelling }}</RouterLink>
                <RouterLink v-if="studyMode === 'hide-spelling'" :to="`/entries/${entry.id}`" class="member-detail" aria-label="查看词条详情"><span aria-hidden="true">↗</span></RouterLink>
              </h3>
              <p :class="{ muted: !entry.definition.trim(), 'masked-definition': studyMode === 'hide-definition' && !!entry.definition.trim() }">
                <StudyText :text="entry.definition.trim() ? entry.definition : '暂未填写释义'" :masked="studyMode === 'hide-definition' && !!entry.definition.trim()" label="释义" kind="definition" />
              </p>
            </div>
            <div class="entry-meta">
              <time :datetime="entry.updatedAt ?? entry.createdAt" :title="'添加时间：' + formatDateTime(entry.createdAt)" :aria-label="'修改时间：' + formatDate(entry.updatedAt ?? entry.createdAt) + '；添加时间：' + formatDateTime(entry.createdAt)">{{ formatDate(entry.updatedAt ?? entry.createdAt) }}</time>
            </div>
          </li>
        </BrowseGrid>
        <p v-if="entries.length === 0" class="browse-empty">没有直接命中的词条。</p>
      </section>
      <section v-if="query" class="search-groups" aria-label="说明命中的关系组">
        <div class="results-heading"><h2>说明命中的关系组 <span class="small-count">{{ groups.length }} 组</span></h2></div>
        <p v-if="groups.length === 0" class="browse-empty">没有匹配的关系说明。</p>
        <BrowseGrid v-else class="group-grid" :items="groups" :layout="layout">
          <GroupCard v-for="group in groups" :key="group.id" :group="group" :mode="studyMode" compact browse />
        </BrowseGrid>
      </section>
    </template>
  </div>
</template>
