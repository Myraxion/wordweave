<script setup lang="ts">
import type { Group } from '../api'
import type { StudyMode } from '../studyMode'
import StudyText from './StudyText.vue'

withDefaults(defineProps<{ group: Group; currentId?: number; mode?: StudyMode; compact?: boolean }>(), {
  mode: 'full',
  compact: false,
})
</script>

<template>
  <article class="relation-card" :class="{ 'compact-group': compact }" :aria-label="mode === 'hide-spelling' ? group.type + '：' + group.members.length + ' 个词条' : group.type + '：' + group.members.map(m => m.spelling).join(' / ')">
    <div v-if="compact" class="group-heading">
      <h3><span class="relation-tag">{{ group.type }}</span><span class="member-count">{{ group.members.length }} 词</span></h3>
      <RouterLink :to="'/groups/' + group.id" class="detail-link" aria-label="编辑关系组">编辑组 <span aria-hidden="true">↗</span></RouterLink>
    </div>
    <h3 v-else class="group-title">
      <RouterLink :to="'/groups/' + group.id">
        {{ mode === 'hide-spelling' ? group.type + ' · ' + group.members.length + ' 个词条' : group.members.map(m => m.spelling).join(' / ') }}
      </RouterLink>
    </h3>
    <ul class="group-members">
      <li v-for="member in group.members" :key="member.id" :class="{ 'current-member': member.id === currentId }">
        <div class="member-heading">
          <StudyText v-if="mode === 'hide-spelling'" class="member-spelling" lang="en" :text="member.spelling" masked label="拼写" kind="spelling" />
          <RouterLink v-else :to="'/entries/' + member.id" class="member-spelling" lang="en">{{ member.spelling }}</RouterLink>
          <span v-if="compact && member.id === currentId" class="current-member-label">当前</span>
          <RouterLink v-if="compact" :to="'/entries/' + member.id" class="member-detail" :aria-label="mode === 'hide-spelling' ? '查看词条详情' : '查看 ' + member.spelling + ' 的详情'"><span aria-hidden="true">↗</span></RouterLink>
        </div>
        <p class="definition" :class="{ muted: !member.definition.trim() }">
          <StudyText :text="member.definition.trim() ? member.definition : '暂未填写释义'" :masked="mode === 'hide-definition' && !!member.definition.trim()" label="释义" kind="definition" />
        </p>
      </li>
    </ul>
    <div v-if="group.note.trim() && mode === 'full'" class="group-note">
      <h4>关系说明</h4>
      <p class="definition">{{ group.note }}</p>
    </div>
    <details v-else-if="group.note.trim()" :key="mode" class="group-note">
      <summary>关系说明 <span>展开查看</span></summary>
      <p class="definition">{{ group.note }}</p>
    </details>
  </article>
</template>
