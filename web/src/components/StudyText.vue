<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{ text: string; masked: boolean; label: string; kind: 'spelling' | 'definition' }>()
const pinned = ref(false)
const hovered = ref(false)
const revealed = computed(() => pinned.value || hovered.value)
watch(() => [props.masked, props.text], () => {
  pinned.value = false
  hovered.value = false
})

function toggle() {
  pinned.value = !pinned.value
  hovered.value = false
}
</script>

<template>
  <button
    v-if="masked"
    type="button"
    class="study-text reveal-text"
    :class="{ concealed: !revealed, 'reveal-spelling': kind === 'spelling' }"
    :aria-label="(pinned ? '收回' : '显示') + label"
    :aria-pressed="pinned"
    @pointerenter="hovered = $event.pointerType === 'mouse'"
    @pointerleave="hovered = false"
    @click="toggle"
  >
    <span class="study-content">{{ text }}</span>
  </button>
  <span v-else class="study-text">{{ text }}</span>
</template>
