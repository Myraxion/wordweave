<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { confirmDiscard, hasUnsavedChanges, useLeaveProtection } from '../leaveProtection'
import GroupCard from '../components/GroupCard.vue'
import GroupForm from '../components/GroupForm.vue'
import {
  ApiError,
  deleteEntry,
  errorMessage,
  formatDate,
  getEntry,
  listEntryGroups,
  updateEntry,
  type Entry,
  type Group,
} from '../api'

const props = defineProps<{ id: string }>()
const router = useRouter()
const entry = ref<Entry>()
const error = ref('')
const loading = ref(true)
const groups = ref<Group[]>([])
const creatingGroup = ref(false)
const confirmingDelete = ref(false)
const deleteError = ref('')
const conflictGroup = ref<Group>()
const deleting = ref(false)

async function startDelete() {
  deleteError.value = ''
  conflictGroup.value = undefined
  try {
    groups.value = await listEntryGroups(props.id)
    confirmingDelete.value = true
  } catch (err) {
    deleteError.value = errorMessage(err)
  }
}

async function removeEntry() {
  deleting.value = true
  deleteError.value = ''
  conflictGroup.value = undefined
  try {
    await deleteEntry(props.id)
    await router.push('/')
  } catch (err) {
    deleteError.value = errorMessage(err)
    if (err instanceof ApiError) conflictGroup.value = err.body.existingGroup
  } finally {
    deleting.value = false
  }
}
const sections = computed(() => [...new Set(groups.value.map(group => group.type))].map(type => ({
  type,
  groups: groups.value.filter(group => group.type === type),
})))

const editing = ref(false)
const formSpelling = ref('')
const formDefinition = ref('')
const saving = ref(false)
const formError = ref('')
const existingEntry = ref<Entry>()
const editButton = ref<HTMLButtonElement>()
const spellingInput = ref<HTMLInputElement>()
const dirty = computed(() => editing.value && !!entry.value && (formSpelling.value !== entry.value.spelling || formDefinition.value !== entry.value.definition))
useLeaveProtection(dirty, saving)

async function load() {
  loading.value = true
  error.value = ''
  editing.value = false
  try {
    const [loadedEntry, loadedGroups] = await Promise.all([getEntry(props.id), listEntryGroups(props.id)])
    entry.value = loadedEntry
    groups.value = loadedGroups
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    loading.value = false
  }
}

function startEditing() {
  if (!entry.value) return
  if (creatingGroup.value && hasUnsavedChanges() && !confirmDiscard()) return
  creatingGroup.value = false
  formSpelling.value = entry.value.spelling
  formDefinition.value = entry.value.definition
  formError.value = ''
  existingEntry.value = undefined
  editing.value = true
  void nextTick(() => spellingInput.value?.focus())
}

function cancelEditing() {
  if (dirty.value && !confirmDiscard()) return
  editing.value = false
  formError.value = ''
  existingEntry.value = undefined
  void nextTick(() => editButton.value?.focus())
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
    groups.value = await listEntryGroups(props.id)
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

async function groupSaved() {
  creatingGroup.value = false
  await load()
}
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

          <h1 class="word-title" lang="en">{{ entry.spelling }}</h1>
          <p class="subtitle">
            收录于
            <time :datetime="entry.createdAt">{{
              formatDate(entry.createdAt)
            }}</time>
          </p>
        </div>
        <div class="heading-actions">
          <button class="button primary" type="button" @click="creatingGroup = true">＋ 建立关系组</button>
          <button ref="editButton" class="button secondary" type="button" @click="startEditing">
            ✎ 编辑词条
          </button>
          <button class="button secondary" type="button" @click="startDelete">删除词条</button>
          <RouterLink class="button secondary" to="/entries/new"
            >＋ 继续收词</RouterLink
          >
        </div>
      </div>
      <div v-if="deleteError" class="notice error" role="alert">
        <p>{{ deleteError }}</p>
        <RouterLink v-if="conflictGroup" :to="`/groups/${conflictGroup.id}`">查看冲突关系组：{{ conflictGroup.type }}（{{ conflictGroup.members.map(member => member.spelling).join('、') }}）→</RouterLink>
      </div>
      <section v-if="confirmingDelete" class="detail-card" aria-label="删除词条确认">
        <h2>删除 {{ entry.spelling }}？</h2>
        <p>删除后将从以下关系组中移除此词条；成员不足两个的关系组会被删除。</p>
        <p v-if="!groups.length">此词条没有关系组。</p>
        <ul v-else>
          <li v-for="group in groups" :key="group.id">
            <RouterLink :to="`/groups/${group.id}`">{{ group.type }}：{{ group.members.map(member => member.spelling).join('、') }}</RouterLink>
            — {{ group.members.length === 2 ? '此组将被删除' : '此组将保留' }}
          </li>
        </ul>
        <div class="form-actions">
          <button class="button secondary" type="button" :disabled="deleting" @click="confirmingDelete = false; deleteError = ''">取消</button>
          <button class="button primary" type="button" :disabled="deleting" @click="removeEntry">{{ deleting ? '正在删除…' : '确认删除' }}</button>
        </div>
      </section>
      <GroupForm v-if="creatingGroup" :entry="entry" @saved="groupSaved" @cancel="creatingGroup = false" />
      <section class="detail-card">
        <h2>释义<span class="section-line"></span></h2>
        <p class="definition" :class="{ muted: !entry.definition }">
          {{ entry.definition || '暂未填写释义' }}
        </p>
      </section>
      <section class="entry-relations" aria-label="词条所属关系组">
        <div class="results-heading">
          <h2>关系组 <span class="small-count">{{ groups.length }}</span></h2>
        </div>
        <div v-if="!groups.length" class="group-empty">
          <span aria-hidden="true">○ — ○</span>
          <p>暂无关系组</p>
          <small>这个词条还没有关联到其他词条。</small>
        </div>
        <section v-for="section in sections" :key="section.type" class="relation-section" :aria-label="section.type">
          <h3 class="relation-type">{{ section.type }} <span class="small-count">{{ section.groups.length }}</span></h3>
          <div class="group-grid">
            <GroupCard v-for="group in section.groups" :key="group.id" :group="group" :current-id="entry.id" compact />
          </div>
        </section>
      </section>
    </div>

    <form v-else class="form-card" @submit.prevent="save">
      <div class="page-heading" style="margin-bottom: 24px">
        <div>

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
          ref="spellingInput"
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
