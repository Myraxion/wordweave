<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError, createEntry, errorMessage, type Entry } from '../api'

const router = useRouter()
const spelling = ref('')
const definition = ref('')
const saving = ref(false)
const error = ref('')
const existing = ref<Entry>()

async function save() {
  error.value = ''
  existing.value = undefined
  if (!spelling.value.trim()) {
    error.value = '请输入词条拼写。'
    return
  }
  saving.value = true
  try {
    const entry = await createEntry(spelling.value, definition.value)
    await router.push(`/entries/${entry.id}`)
  } catch (err) {
    error.value = errorMessage(err)
    if (err instanceof ApiError) existing.value = err.body.existingEntry
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <RouterLink class="back-link" to="/">← 返回全部词条</RouterLink>
  <div class="page-heading">
    <div>
      <p class="eyebrow">A NEW WORD, A NEW CONNECTION</p>
      <h1>收录词条</h1>
      <p class="subtitle">先留住这个词，再一点点丰富你的理解。</p>
    </div>
  </div>
  <form class="form-card" @submit.prevent="save">
    <div class="field">
      <label for="spelling">词条拼写 <span class="required">必填</span></label
      ><input
        id="spelling"
        v-model="spelling"
        name="spelling"
        placeholder="例如 Apple、give up、mother-in-law"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        :disabled="saving"
        :aria-invalid="!!error && !existing"
        :aria-describedby="error ? 'save-error' : 'spelling-hint'"
      />
      <p id="spelling-hint" class="field-hint">
        支持英文单词和短语，保存时会去掉首尾空格。
      </p>
    </div>
    <div class="field">
      <label for="definition">释义 <span class="optional">选填</span></label
      ><textarea
        id="definition"
        v-model="definition"
        name="definition"
        rows="7"
        placeholder="写下它的含义、用法，或你遇见它的语境…"
        :disabled="saving"
      ></textarea>
      <p class="field-hint">支持多行纯文本。还没想好？留空也可以。</p>
    </div>
    <div v-if="error" id="save-error" class="notice error" role="alert">
      <p>{{ error }}</p>
      <RouterLink v-if="existing" :to="`/entries/${existing.id}`"
        >查看已有词条：{{ existing.spelling }} →</RouterLink
      >
    </div>
    <div class="form-actions">
      <RouterLink class="button secondary" to="/">取消</RouterLink
      ><button class="button primary" type="submit" :disabled="saving">
        {{ saving ? '正在保存…' : '保存词条' }}
        <span aria-hidden="true">→</span>
      </button>
    </div>
  </form>
</template>
