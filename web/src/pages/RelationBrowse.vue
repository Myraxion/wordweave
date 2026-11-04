<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { errorMessage, getGroupTypes, listGroupsByType, type Group } from '../api'
import DisplayModeSwitch from '../components/DisplayModeSwitch.vue'
import DateGrid from '../components/DateGrid.vue'
import GroupCard from '../components/GroupCard.vue'
import LayoutSwitch from '../components/LayoutSwitch.vue'
import { useBrowseLayout } from '../browseLayout'
import { studyMode } from '../studyMode'

const types = ref<string[]>([])
const route = useRoute()
const router = useRouter()
const selected = ref(typeof route.query.type === 'string' ? route.query.type : '')
const sortOptions = [
  { value: 'updated_desc', label: '最近修改在前' },
  { value: 'updated_asc', label: '最早修改在前' },
  { value: 'created_desc', label: '最近创建在前' },
  { value: 'created_asc', label: '最早创建在前' },
]
const initialSort = typeof route.query.sort === 'string' ? route.query.sort : 'updated_desc'
const sortOrder = ref(sortOptions.some(option => option.value === initialSort) ? initialSort : 'updated_desc')
const groups = ref<Group[]>([])
const error = ref('')
const loading = ref(true)
const layout = useBrowseLayout('wordweave.relation-layout')
let requestNumber = 0

function changeFilters() {
  void router.replace({ path: '/relations', query: {
    ...(selected.value ? { type: selected.value } : {}),
    ...(sortOrder.value !== 'updated_desc' ? { sort: sortOrder.value } : {}),
  } })
}

async function selectType(type: string) {
  selected.value = type
  const current = ++requestNumber
  loading.value = true
  error.value = ''
  try {
    const result = await listGroupsByType(type, sortOrder.value)
    if (current === requestNumber) groups.value = result
  } catch (err) {
    if (current === requestNumber) error.value = errorMessage(err)
  } finally {
    if (current === requestNumber) loading.value = false
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    types.value = await getGroupTypes()
    await selectType(selected.value)
  } catch (err) {
    error.value = errorMessage(err)
    loading.value = false
  }
}
onMounted(load)
</script>

<template>
  <div class="browse-page">
    <div class="page-heading">
      <div>
        <p class="eyebrow">RELATION GROUPS</p>
        <h1>关系浏览</h1>
        <p class="subtitle">把有关联的词放在一起，对照着记。</p>
      </div>
      <RouterLink class="button primary" :to="{ path: '/groups/new', query: selected ? { type: selected } : {} }"><span aria-hidden="true">＋</span> 建立关系</RouterLink>
    </div>
    <div class="browse-toolbar">
      <div class="browse-options">
        <DisplayModeSwitch />
        <div class="type-filter">
          <label for="relation-type" class="sr-only">关系类型</label>
          <select id="relation-type" v-model="selected" class="sort-select" @change="changeFilters">
            <option value="">所有关系类型</option>
            <option v-for="type in types" :key="type" :value="type">{{ type }}</option>
          </select>
        </div>
      </div>
      <div class="browse-options">
        <LayoutSwitch v-model="layout" label="关系组布局" />
        <label for="group-sort" class="sr-only">关系组排序</label>
        <select id="group-sort" v-model="sortOrder" class="sort-select" @change="changeFilters">
          <option v-for="option in sortOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </div>
    </div>
    <div v-if="error" class="notice error" role="alert">{{ error }} <button class="text-button" @click="load">重试</button></div>
    <p v-else-if="loading" class="loading" role="status">正在读取关系组…</p>
    <section v-else aria-label="关系组列表">
      <div class="results-heading">
        <h2>{{ selected || '全部关系组' }} <span class="small-count">{{ groups.length }} 组</span></h2>
      </div>
      <p v-if="!groups.length" class="browse-empty">{{ selected ? '这个类型还没有关系组，点击“建立关系”开始整理。' : '还没有关系组，点击“建立关系”开始整理。' }}</p>
      <DateGrid v-else :items="groups" :layout="layout" :sort="sortOrder" kind="group" v-slot="{ item: group, grouped }">
        <GroupCard :group="group" :mode="studyMode" :hide-date="grouped" compact browse />
      </DateGrid>
    </section>
  </div>
</template>
