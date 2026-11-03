<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{ text: string; masked: boolean; label: string }>()
const revealed = ref(false)
watch(() => [props.masked, props.text], () => { revealed.value = false })
</script>

<template>
  <button
    v-if="masked"
    type="button"
    class="study-text reveal-text"
    :class="{ concealed: !revealed }"
    :aria-label="(revealed ? '收回' : '显示') + label"
    :aria-pressed="revealed"
    @click="revealed = !revealed"
  >{{ revealed ? text : '点击显示' + label }}</button>
  <span v-else class="study-text">{{ text }}</span>
</template>
