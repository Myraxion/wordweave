<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage, getGroupTypes, listGroupsByType, type Group } from '../api'
import GroupCard from '../components/GroupCard.vue'

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
  try {
    types.value = await getGroupTypes()
    if (types.value[0]) await selectType(types.value[0])
    else loading.value = false
  } catch (err) {
    error.value = errorMessage(err)
    loading.value = false
  }
}
onMounted(load)
</script>

<template>
  <div class="page-heading"><div><p class="eyebrow">RELATION GROUPS</p><h1>按关系类型浏览</h1><p class="subtitle">选择一种关系，集中阅读其中的对照内容。</p></div></div>
  <nav class="type-buttons" aria-label="关系类型">
    <button v-for="type in types" :key="type" class="button secondary" :class="{ selected: type === selected }" :aria-pressed="type === selected" @click="selectType(type)">{{ type }}</button>
  </nav>
  <div v-if="error" class="notice error" role="alert">{{ error }} <button class="text-button" @click="types.length ? selectType(selected) : load()">重试</button></div>
  <p v-else-if="loading" class="loading" role="status">正在读取关系组…</p>
  <section v-else class="detail-card">
    <h2>{{ selected }} <span class="small-count">{{ groups.length }}</span></h2>
    <p v-if="!groups.length" class="muted">这个类型还没有关系组。</p>
    <GroupCard v-for="group in groups" :key="group.id" :group="group" />
  </section>
</template>
