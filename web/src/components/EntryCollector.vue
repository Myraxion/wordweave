<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { confirmDiscard, protectedWrite } from '../leaveProtection'
import { ApiError, createEntry, errorMessage, type Entry } from '../api'

const props = defineProps<{ disabled?: boolean; inline?: boolean; initialSpelling?: string }>()
const emit = defineEmits<{ selected: [entry: Entry]; busy: [value: boolean]; draft: [value: boolean]; cancel: []; finish: [] }>()
const spelling = ref(props.initialSpelling || '')
const definition = ref('')
const saving = ref(false)
const error = ref('')
const existing = ref<Entry>()
const saved = ref<Entry>()
const reused = ref(false)
const spellingInput = ref<HTMLInputElement>()
const hasDraft = computed(() => !!(spelling.value || definition.value))
watch(hasDraft, value => emit('draft', value), { immediate: true })
watch(spelling, () => { error.value = ''; existing.value = undefined })

function clear() {
  spelling.value = ''
  definition.value = ''
  error.value = ''
  existing.value = undefined
}

function discard() {
  if (hasDraft.value && !confirmDiscard()) return
  if (props.inline) emit('cancel')
  else clear()
}

function select(entry: Entry, existingEntry = false) {
  saved.value = entry
  reused.value = existingEntry
  clear()
  emit('selected', entry)
  void nextTick(() => spellingInput.value?.focus())
}

async function save() {
  error.value = ''
  existing.value = undefined
  if (!spelling.value.trim()) {
    error.value = '请输入词条拼写。'
    return
  }
  emit('busy', true)
  let collected: Entry
  try {
    collected = await protectedWrite(saving, () => createEntry(spelling.value, definition.value), () => {})
  } catch (err) {
    error.value = errorMessage(err)
    if (err instanceof ApiError) existing.value = err.body.existingEntry
    return
  } finally {
    emit('busy', false)
  }
  select(collected)
}
</script>

<template>
  <section class="entry-collector" aria-labelledby="collect-entry-heading">
    <h2 id="collect-entry-heading">{{ inline ? '新增缺少的词条' : '收录词条' }}</h2>
    <p v-if="inline" class="field-hint">词条会独立保存，取消建组或编辑后仍保留。</p>
    <p v-else class="subtitle collection-feedback" :class="{ 'is-saved': !!saved }" role="status" :tabindex="saved ? 0 : undefined">
      <template v-if="saved">{{ reused ? '已选择已有词条' : '已收录' }} {{ saved.spelling }}，并加入待建关系组；词条已保存。</template>
      <template v-else>词条会独立保存，并加入待建关系组；不建立关系也可以。</template>
    </p>
    <fieldset :disabled="disabled || saving">
      <div class="field">
        <label for="new-member-spelling">词条拼写 <span class="required">必填</span></label>
        <input id="new-member-spelling" ref="spellingInput" v-model="spelling" autocomplete="off" autocapitalize="off" spellcheck="false" placeholder="例如 Apple、give up、mother-in-law" @keydown.enter.prevent="save" />
      </div>
      <div class="field">
        <label for="new-member-definition">释义 <span class="optional">选填</span></label>
        <textarea id="new-member-definition" v-model="definition" rows="3" placeholder="记录含义、用法或遇见它的语境，暂时留空也可以。"></textarea>
      </div>
      <div v-if="error" class="notice error" role="alert">
        <p>{{ error }}</p>
        <button v-if="existing" type="button" class="text-button" @click="select(existing, true)">选择已有词条：{{ existing.spelling }}</button>
      </div>
      <div class="form-actions">
        <button v-if="!inline" type="button" class="button secondary finish-collection" @click="emit('finish')">结束收录</button>
        <button v-if="inline" type="button" class="button secondary" @click="discard">取消新增</button>
        <button v-else type="button" class="button secondary" :disabled="!hasDraft" @click="discard">清空输入</button>
        <button type="button" class="button primary" @click="save">{{ saving ? '正在收录…' : '收录词条' }}</button>
      </div>
    </fieldset>
  </section>
</template>
