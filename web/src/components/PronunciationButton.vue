<script setup lang="ts">
import { playSpeech } from '../speech'
defineProps<{ spelling: string }>()
</script>

<template>
  <button type="button" class="pronunciation-button" :aria-label="'播放 ' + spelling + ' 的发音'" title="播放发音" @click="playSpeech(spelling)">
    <svg aria-hidden="true" viewBox="0 0 24 24" width="19" height="19" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M4 9v6h4l5 4V5L8 9H4z"/><path d="M16 9a4 4 0 0 1 0 6M19 6a8 8 0 0 1 0 12"/></svg>
  </button>
</template>
