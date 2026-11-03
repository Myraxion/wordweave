<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { deleteGroup, errorMessage, getGroup, type Group } from '../api'
import GroupCard from '../components/GroupCard.vue'
import GroupForm from '../components/GroupForm.vue'

const props = defineProps<{ id: string }>()
const group = ref<Group>()
const loading = ref(true)
const error = ref('')
const editing = ref(false)
const deleting = ref(false)
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
watch(() => props.id, () => { editing.value = false; void load() }, { immediate: true })

async function remove() {
  if (!group.value || !window.confirm('确定删除整个关系组吗？所有成员词条都会保留。')) return
  deleting.value = true
  try {
    await deleteGroup(group.value.id)
    await router.push('/')
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    deleting.value = false
  }
}

function saved(updated: Group) {
  group.value = updated
  editing.value = false
}
</script>

<template>
  <RouterLink class="back-link" to="/relations">← 返回关系类型浏览</RouterLink>
  <div v-if="error" class="notice error" role="alert">{{ error }} <button class="text-button" @click="load">重试</button></div>
  <p v-else-if="loading" class="loading" role="status">正在读取关系组…</p>
  <template v-else-if="group">
    <div class="page-heading">
      <div><p class="eyebrow">READ TOGETHER</p><h1>{{ group.type }}</h1><p class="subtitle">点击成员，继续查看词条及其关系组。</p></div>
      <div v-if="!editing" class="heading-actions">
        <button class="button primary" type="button" @click="editing = true">编辑关系组</button>
        <button class="button secondary" type="button" :disabled="deleting" @click="remove">删除关系组</button>
      </div>
    </div>
    <GroupForm v-if="editing" :group="group" @saved="saved" @deleted="router.push('/')" @cancel="editing = false" />
    <GroupCard v-else :group="group" />
  </template>
</template>
