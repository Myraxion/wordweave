<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage, getGroup, type Group } from '../api'
import GroupCard from '../components/GroupCard.vue'

const props = defineProps<{ id: string }>()
const group = ref<Group>()
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    group.value = await getGroup(props.id)
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
  <div v-if="error" class="notice error" role="alert">{{ error }} <button class="text-button" @click="load">重试</button></div>
  <p v-else-if="loading" class="loading" role="status">正在读取关系组…</p>
  <template v-else-if="group">
    <div class="page-heading">
      <div><p class="eyebrow">READ TOGETHER</p><h1>{{ group.type }}</h1><p class="subtitle">点击成员，继续查看词条及其关系组。</p></div>
    </div>
    <GroupCard :group="group" />
  </template>
</template>
