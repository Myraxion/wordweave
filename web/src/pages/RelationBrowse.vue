<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage, getGroupTypes, listGroupsByType, type Group } from '../api'
import DisplayModeSwitch from '../components/DisplayModeSwitch.vue'
import GroupCard from '../components/GroupCard.vue'
import { studyMode } from '../studyMode'

const types = ref<string[]>([])
const selected = ref('')
const groups = ref<Group[]>([])
const error = ref('')
const loading = ref(true)
let requestNumber = 0

async function selectType(type: string) {
  selected.value = type
  const current = ++requestNumber
  loading.value = true
  error.value = ''
  try {
    const result = await listGroupsByType(type)
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
        <h1>关系类型浏览</h1>
        <p class="subtitle">把有关联的词放在一起，对照着记。</p>
      </div>
    </div>
    <div class="browse-toolbar">
      <div class="type-filter">
        <label for="relation-type" class="sr-only">关系类型</label>
        <select id="relation-type" :value="selected" class="sort-select" @change="selectType(($event.target as HTMLSelectElement).value)">
          <option value="">所有关系类型（全部）</option>
          <option v-for="type in types" :key="type" :value="type">{{ type }}</option>
        </select>
      </div>
      <DisplayModeSwitch />
    </div>
    <div v-if="error" class="notice error" role="alert">{{ error }} <button class="text-button" @click="load">重试</button></div>
    <p v-else-if="loading" class="loading" role="status">正在读取关系组…</p>
    <section v-else aria-label="关系组列表">
      <div class="results-heading">
        <h2>{{ selected || '全部关系组' }} <span class="small-count">{{ groups.length }} 组</span></h2>
        <span class="results-caption">最近创建在前</span>
      </div>
      <p v-if="!groups.length" class="browse-empty">{{ selected ? '这个类型还没有关系组。' : '还没有关系组，可以从词条详情开始建立。' }}</p>
      <div v-else class="group-grid">
        <GroupCard v-for="group in groups" :key="group.id" :group="group" :mode="studyMode" compact />
      </div>
    </section>
  </div>
</template>
