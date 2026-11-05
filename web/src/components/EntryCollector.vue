<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { ApiError, createEntry, errorMessage, type Entry } from '../api'

const props = defineProps<{ disabled?: boolean; inline?: boolean; initialSpelling?: string }>()
const emit = defineEmits<{ selected: [entry: Entry]; busy: [value: boolean]; draft: [value: boolean]; cancel: [] }>()
const spelling = ref(props.initialSpelling || '')
const definition = ref('')
const saving = ref(false)
const error = ref('')
const existing = ref<Entry>()
const saved = ref<Entry>()
const spellingInput = ref<HTMLInputElement>()
const hasDraft = computed(() => !!(spelling.value.trim() || definition.value.trim()))
watch(hasDraft, value => emit('draft', value), { immediate: true })
watch(spelling, () => { error.value = ''; existing.value = undefined })

function clear() {
  spelling.value = ''
  definition.value = ''
  error.value = ''
  existing.value = undefined
}

function select(entry: Entry) {
  saved.value = entry
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
  saving.value = true
  emit('busy', true)
  try {
    select(await createEntry(spelling.value, definition.value))
  } catch (err) {
    error.value = errorMessage(err)
    if (err instanceof ApiError) existing.value = err.body.existingEntry
  } finally {
    saving.value = false
    emit('busy', false)
  }
}
</script>

<template>
  <section class="entry-collector" aria-labelledby="collect-entry-heading">
    <h2 id="collect-entry-heading">{{ inline ? '新增缺少的词条' : '收录词条' }}</h2>
    <p class="field-hint">{{ inline ? '词条会独立保存，取消建组或编辑后仍保留。' : '词条会独立保存，并加入待建关系组；不建立关系也可以。' }}</p>
    <p v-if="saved && !inline" class="collection-feedback" role="status">已选择 {{ saved.spelling }}，可继续收词或建立关系组。</p>
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
        <button v-if="existing" type="button" class="text-button" @click="select(existing)">选择已有词条：{{ existing.spelling }}</button>
      </div>
      <div class="form-actions">
        <button v-if="inline" type="button" class="button secondary" @click="emit('cancel')">取消新增</button>
        <button v-else type="button" class="button secondary" :disabled="!hasDraft" @click="clear">清空输入</button>
        <button type="button" class="button primary" @click="save">{{ saving ? '正在收录…' : '收录词条' }}</button>
      </div>
    </fieldset>
  </section>
</template>
