<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  ApiError,
  errorMessage,
  formatDate,
  getEntry,
  updateEntry,
  type Entry,
} from '../api'

const props = defineProps<{ id: string }>()
const entry = ref<Entry>()
const error = ref('')
const loading = ref(true)

const editing = ref(false)
const formSpelling = ref('')
const formDefinition = ref('')
const saving = ref(false)
const formError = ref('')
const existingEntry = ref<Entry>()

async function load() {
  loading.value = true
  error.value = ''
  editing.value = false
  try {
    entry.value = await getEntry(props.id)
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    loading.value = false
  }
}

function startEditing() {
  if (!entry.value) return
  formSpelling.value = entry.value.spelling
  formDefinition.value = entry.value.definition
  formError.value = ''
  existingEntry.value = undefined
  editing.value = true
}

function cancelEditing() {
  editing.value = false
  formError.value = ''
  existingEntry.value = undefined
}

async function save() {
  if (!entry.value) return
  formError.value = ''
  existingEntry.value = undefined
  if (!formSpelling.value.trim()) {
    formError.value = '请输入词条拼写。'
    return
  }
  saving.value = true
  try {
    const updated = await updateEntry(
      props.id,
      formSpelling.value,
      formDefinition.value,
    )
    entry.value = updated
    editing.value = false
  } catch (err) {
    formError.value = errorMessage(err)
    if (err instanceof ApiError) {
      existingEntry.value = err.body.existingEntry
    }
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <RouterLink class="back-link" to="/">← 返回全部词条</RouterLink>
  <div v-if="error" class="notice error" role="alert">
    {{ error }} <button class="text-button" @click="load">重试</button>
  </div>
  <p v-else-if="loading" role="status" class="loading">正在读取词条…</p>
  <template v-else-if="entry">
    <div v-if="!editing">
      <div class="page-heading">
        <div>
          <p class="eyebrow">IN YOUR COLLECTION</p>
          <h1 class="word-title" lang="en">{{ entry.spelling }}</h1>
          <p class="subtitle">
            收录于
            <time :datetime="entry.createdAt">{{
              formatDate(entry.createdAt)
            }}</time>
          </p>
        </div>
        <div class="heading-actions">
          <button class="button secondary" type="button" @click="startEditing">
            ✎ 编辑词条
          </button>
          <RouterLink class="button secondary" to="/entries/new"
            >＋ 继续收词</RouterLink
          >
        </div>
      </div>
      <section class="detail-card">
        <h2>释义<span class="section-line"></span></h2>
        <p class="definition" :class="{ muted: !entry.definition }">
          {{ entry.definition || '暂未填写释义' }}
        </p>
      </section>
      <section class="detail-card">
        <h2>
          关系组<span class="small-count">0</span
          ><span class="section-line"></span>
        </h2>
        <div class="group-empty">
          <span aria-hidden="true">○ — ○</span>
          <p>暂无关系组</p>
          <small>这个词条还没有关联到其他词条。</small>
        </div>
      </section>
    </div>

    <form v-else class="form-card" @submit.prevent="save">
      <div class="page-heading" style="margin-bottom: 24px">
        <div>
          <p class="eyebrow">EDIT WORD</p>
          <h1>编辑词条</h1>
          <p class="subtitle">
            收录于
            <time :datetime="entry.createdAt">{{
              formatDate(entry.createdAt)
            }}</time>
          </p>
        </div>
      </div>
      <div class="field">
        <label for="spelling">词条拼写 <span class="required">必填</span></label
        ><input
          id="spelling"
          v-model="formSpelling"
          name="spelling"
          placeholder="例如 Apple、give up、mother-in-law"
          autocomplete="off"
          autocapitalize="off"
          spellcheck="false"
          :disabled="saving"
          :aria-invalid="!!formError && !existingEntry"
          :aria-describedby="formError ? 'edit-error' : 'spelling-hint'"
        />
        <p id="spelling-hint" class="field-hint">
          支持英文单词和短语，保存时会去掉首尾空格。
        </p>
      </div>
      <div class="field">
        <label for="definition">释义 <span class="optional">选填</span></label
        ><textarea
          id="definition"
          v-model="formDefinition"
          name="definition"
          rows="7"
          placeholder="写下它的含义、用法，或你遇见它的语境…"
          :disabled="saving"
        ></textarea>
        <p class="field-hint">支持多行纯文本。可清空释义。</p>
      </div>
      <div v-if="formError" id="edit-error" class="notice error" role="alert">
        <p>{{ formError }}</p>
        <RouterLink
          v-if="existingEntry"
          :to="`/entries/${existingEntry.id}`"
        >
          查看已有词条：{{ existingEntry.spelling }} →
        </RouterLink>
      </div>
      <div class="form-actions">
        <button
          class="button secondary"
          type="button"
          :disabled="saving"
          @click="cancelEditing"
        >
          取消</button
        ><button class="button primary" type="submit" :disabled="saving">
          {{ saving ? '正在保存…' : '保存修改' }}
          <span aria-hidden="true">→</span>
        </button>
      </div>
    </form>
  </template>
</template>
