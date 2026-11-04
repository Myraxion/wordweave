<script setup lang="ts">
import { computed, nextTick, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import GroupForm from '../components/GroupForm.vue'

const route = useRoute()
const router = useRouter()
const fromRelations = computed(() => route.path === '/groups/new')
const initialType = computed(() => typeof route.query.type === 'string' ? route.query.type : '')
const returnTo = computed(() => fromRelations.value
  ? { path: '/relations', query: initialType.value ? { type: initialType.value } : {} }
  : { path: '/' })

onMounted(async () => {
  await nextTick()
  document.getElementById(fromRelations.value ? 'member-search' : 'new-member-spelling')?.focus({ preventScroll: true })
})
</script>

<template>
  <RouterLink class="back-link" :to="returnTo">← 返回{{ fromRelations ? '关系浏览' : '全部词条' }}</RouterLink>
  <div class="page-heading">
    <div>
      <h1>收录与建组</h1>
      <p class="subtitle">收录新词，或选择已有词条一起整理关系。</p>
    </div>
  </div>
  <GroupForm collecting :initial-type="initialType" @saved="group => router.push('/groups/' + group.id)" @cancel="router.push(returnTo)" />
</template>
