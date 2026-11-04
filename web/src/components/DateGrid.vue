<script setup lang="ts" generic="T extends { id: number; createdAt: string; updatedAt: string | null }">
import { computed } from 'vue'
import type { BrowseLayout } from '../browseLayout'
import BrowseGrid from './BrowseGrid.vue'

const props = defineProps<{ items: T[]; layout: BrowseLayout; sort: string; kind: 'entry' | 'group'; sortItems?: boolean }>()
const grouped = computed(() => props.layout === 'grid' && !props.sort.startsWith('alpha_'))
const dateValue = (item: T) => props.sort.startsWith('created_') ? item.createdAt : (item.updatedAt ?? item.createdAt)
const formatter = new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
const groups = computed(() => {
  const items = [...props.items]
  if (props.sortItems) {
    const direction = props.sort.endsWith('_asc') ? 1 : -1
    items.sort((a, b) => {
      const left = dateValue(a)
      const right = dateValue(b)
      const difference = Date.parse(left) - Date.parse(right)
      if (difference) return direction * difference
      // The API timestamps can include nanoseconds beyond JavaScript's milliseconds.
      const fraction = (value: string) => (value.match(/\.(\d+)/)?.[1] ?? '').padEnd(9, '0')
      return direction * (fraction(left).localeCompare(fraction(right)) || a.id - b.id)
    })
  }
  const result: { date: string; items: T[] }[] = []
  for (const item of items) {
    const date = formatter.format(new Date(dateValue(item)))
    const previous = result[result.length - 1]
    if (previous?.date === date) previous.items.push(item)
    else result.push({ date, items: [item] })
  }
  return result
})
</script>

<template>
  <div v-if="grouped && items.length" class="date-timeline">
    <p class="timeline-basis">{{ sort.startsWith('created_') ? (kind === 'entry' ? '按添加日期' : '按创建日期') : '按修改日期' }}</p>
    <section v-for="group in groups" :key="group.date" class="timeline-group">
      <h3 class="timeline-date">{{ group.date }}</h3>
      <component :is="kind === 'entry' ? 'ul' : 'div'" :class="kind === 'entry' ? 'entry-list compact-entry-list' : 'group-grid'">
        <template v-for="item in group.items" :key="item.id"><slot :item="item" :grouped="true" /></template>
      </component>
    </section>
  </div>
  <BrowseGrid v-else :as="kind === 'entry' ? 'ul' : 'div'" :class="kind === 'entry' ? 'entry-list compact-entry-list' : 'group-grid'" :items="items" :layout="layout">
    <template v-for="item in items" :key="item.id"><slot :item="item" :grouped="false" /></template>
  </BrowseGrid>
</template>
