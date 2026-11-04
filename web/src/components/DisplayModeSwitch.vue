<script setup lang="ts">
import { studyMode, studyModes } from '../studyMode'
</script>

<template>
  <div class="display-mode-control">
    <div class="display-modes" role="group" aria-label="展示模式">
      <button
        v-for="mode in studyModes"
        :key="mode.value"
        type="button"
        :aria-pressed="studyMode === mode.value"
        @click="studyMode = mode.value"
      >{{ mode.label }}</button>
    </div>
  </div>
</template>
