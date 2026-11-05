<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'

const props = defineProps<{ text: string; masked: boolean; label: string; kind: 'spelling' | 'definition' }>()
const pinned = ref(false)
const hovered = ref(false)
const revealed = computed(() => pinned.value || hovered.value)
const answerId = useId()
const toggleButton = ref<HTMLButtonElement | null>(null)
const answer = ref<HTMLElement | null>(null)
watch(() => [props.masked, props.text], () => {
  pinned.value = false
  hovered.value = false
})

function toggle() {
  if (pinned.value && answer.value?.contains(document.activeElement)) toggleButton.value?.focus()
  pinned.value = !pinned.value
  hovered.value = false
}

function leave() {
  if (!pinned.value && answer.value?.contains(document.activeElement)) toggleButton.value?.focus()
  hovered.value = false
}
</script>

<template>
  <span
    v-if="masked"
    class="study-text reveal-text"
    :class="{ concealed: !revealed, 'reveal-spelling': kind === 'spelling' }"
    @pointerenter="hovered = $event.pointerType === 'mouse'"
    @pointerleave="leave"
  >
    <button ref="toggleButton" type="button" class="reveal-toggle" :aria-label="(pinned ? '收回' : '显示') + label"
      :aria-pressed="pinned" :aria-describedby="revealed ? answerId : undefined" @click="toggle" />
    <span v-if="revealed" :id="answerId" ref="answer" class="study-content" tabindex="0" role="region"
      :aria-label="label" @click="toggle">{{ text }}</span>
  </span>
  <span v-else class="study-text">{{ text }}</span>
</template>
