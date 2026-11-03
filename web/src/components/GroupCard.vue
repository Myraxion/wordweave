<script setup lang="ts">
import type { Group } from '../api'

defineProps<{ group: Group; currentId?: number }>()
</script>

<template>
  <article class="relation-card" :aria-label="group.type + '：' + group.members.map(m => m.spelling).join(' / ')">
    <h3 class="group-title">
      <RouterLink :to="'/groups/' + group.id">
        {{ group.members.map(m => m.spelling).join(' / ') }}
      </RouterLink>
    </h3>
    <ul class="group-members">
      <li v-for="member in group.members" :key="member.id" :class="{ 'current-member': member.id === currentId }">
        <RouterLink :to="'/entries/' + member.id" class="member-spelling" lang="en">{{ member.spelling }}</RouterLink>
        <p class="definition" :class="{ muted: !member.definition }">{{ member.definition || '暂未填写释义' }}</p>
      </li>
    </ul>
    <div class="group-note">
      <h4>关系说明</h4>
      <p class="definition" :class="{ muted: !group.note }">{{ group.note || '暂未填写关系说明' }}</p>
    </div>
  </article>
</template>
