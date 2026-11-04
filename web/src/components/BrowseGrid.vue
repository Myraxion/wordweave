<script setup lang="ts">
import { ref, watchPostEffect } from 'vue'
import type { BrowseLayout } from '../browseLayout'

const props = withDefaults(defineProps<{ items: readonly unknown[]; layout: BrowseLayout; as?: 'div' | 'ul' }>(), { as: 'div' })
const grid = ref<HTMLElement>()

watchPostEffect(onCleanup => {
  // Reconnect when filtering replaces the cards, even if their count stays the same.
  const items = props.items
  const element = grid.value
  if (!element || props.layout !== 'masonry' || !items.length) return

  const cards = Array.from(element.children) as HTMLElement[]
  let frame = 0

  function arrange() {
    // Clear placements first so old columns cannot become implicit columns after resizing.
    for (const card of cards) {
      card.style.removeProperty('grid-column')
      card.style.removeProperty('grid-row')
    }
    const style = getComputedStyle(element!)
    const columns = style.gridTemplateColumns.split(' ').length
    const gap = parseFloat(style.columnGap)
    const bottoms = Array<number>(columns).fill(0)

    // Place cards in source order into the shortest column without moving DOM nodes.
    for (const card of cards) {
      const column = bottoms.indexOf(Math.min(...bottoms))
      const height = Math.ceil(card.getBoundingClientRect().height)
      card.style.gridColumn = String(column + 1)
      card.style.gridRow = `${bottoms[column]! + 1} / span ${height}`
      bottoms[column] = bottoms[column]! + height + gap
    }
  }

  const observer = new ResizeObserver(() => {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(arrange)
  })
  observer.observe(element)
  cards.forEach(card => observer.observe(card))
  arrange()

  onCleanup(() => {
    observer.disconnect()
    cancelAnimationFrame(frame)
    for (const card of cards) {
      card.style.removeProperty('grid-column')
      card.style.removeProperty('grid-row')
    }
  })
})
</script>

<template>
  <component :is="as" ref="grid" :class="{ 'masonry-grid': layout === 'masonry' }">
    <slot />
  </component>
</template>
