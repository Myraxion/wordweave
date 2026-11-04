<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { errorMessage, getGroup, type Group } from '../api'
import GroupForm from '../components/GroupForm.vue'

const props = defineProps<{ id: string }>()
const group = ref<Group>()
const loading = ref(true)
const error = ref('')
const router = useRouter()

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
watch(() => props.id, load, { immediate: true })
</script>

<template>
  <RouterLink class="back-link" to="/relations">← 返回关系浏览</RouterLink>
  <div v-if="error" class="notice error" role="alert">{{ error }} <button class="text-button" @click="load">重试</button></div>
  <p v-else-if="loading" class="loading" role="status">正在读取关系组…</p>
  <GroupForm v-else-if="group" standalone :group="group" @saved="router.push('/relations')" @deleted="router.push('/relations')" @cancel="router.push('/relations')" />
</template>
