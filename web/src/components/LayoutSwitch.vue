<script setup lang="ts">
import type { BrowseLayout } from '../browseLayout'

defineProps<{ label: string }>()
const layout = defineModel<BrowseLayout>({ required: true })
const layouts = [
  { value: 'grid', label: '网格' },
  { value: 'masonry', label: '瀑布流' },
] as const
</script>

<template>
  <div class="display-modes" role="group" :aria-label="label">
    <button
      v-for="option in layouts"
      :key="option.value"
      type="button"
      :aria-pressed="layout === option.value"
      @click="layout = option.value"
    >{{ option.label }}</button>
  </div>
</template>
