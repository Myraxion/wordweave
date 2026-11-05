<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { ApiError, createGroup, deleteGroup, errorMessage, getGroupTypes, listEntries, updateGroup, type Entry, type Group } from '../api'
import { confirmDiscard, useLeaveProtection } from '../leaveProtection'
import EntryCollector from './EntryCollector.vue'

const props = defineProps<{ entry?: Entry; group?: Group; initialType?: string; standalone?: boolean; collecting?: boolean }>()
const emit = defineEmits<{ saved: [group: Group]; deleted: []; cancel: [] }>()
const types = ref<string[]>([])
const initialType = ref(props.group?.type || props.initialType || '')
const type = ref(initialType.value)
const note = ref(props.group?.note || '')
const query = ref('')
const results = ref<Entry[]>([])
const selected = ref<Entry[]>(props.group ? [...props.group.members] : [])
const loading = ref(false)
const searchError = ref('')
const typesError = ref('')
const saving = ref(false)
const deleting = ref(false)
const busy = computed(() => saving.value || entrySaving.value || deleting.value)
const error = ref('')
const existingGroup = ref<Group>()
const addingEntry = ref(false)
const entrySaving = ref(false)
const collectionDraft = ref(false)
const inlineDraft = ref(false)
const dirty = computed(() => {
  const baseline = props.group?.members.map(member => member.id).sort().join(',') || ''
  const members = selected.value.map(member => member.id).sort().join(',')
  return collectionDraft.value || inlineDraft.value || members !== baseline || type.value !== initialType.value || note.value !== (props.group?.note || '')
})
useLeaveProtection(dirty, busy)
const choosingMembers = ref(false)
const memberList = ref<HTMLElement>()
const memberListEntry = ref<HTMLElement>()
const candidates = computed(() => results.value.filter(member => member.id !== props.entry?.id))
const memberCount = computed(() => selected.value.length + (props.entry ? 1 : 0))
const saveHint = computed(() => {
  if (addingEntry.value) return '请先完成或取消新增词条。'
  if (collectionDraft.value) return '请先收录或清空未保存的词条。'
  if (!props.group && memberCount.value < 2) return '请至少选择两个词条。'
  if (!type.value) return '请选择关系类型。'
  return ''
})
let searchNumber = 0

async function loadTypes() {
  typesError.value = ''
  try {
    types.value = await getGroupTypes()
    if (!types.value.includes(type.value)) {
      if (type.value === initialType.value) initialType.value = ''
      type.value = ''
    }
  } catch (err) {
    typesError.value = errorMessage(err)
  }
}

async function search() {
  const current = ++searchNumber
  loading.value = true
  searchError.value = ''
  try {
    const found = await listEntries('alpha_asc', query.value)
    if (current === searchNumber) results.value = found
  } catch (err) {
    if (current === searchNumber) searchError.value = errorMessage(err)
  } finally {
    if (current === searchNumber) loading.value = false
  }
}

function toggle(member: Entry) {
  const index = selected.value.findIndex(value => value.id === member.id)
  if (index < 0) selected.value.push(member)
  else selected.value.splice(index, 1)
}

async function enterMemberList() {
  choosingMembers.value = true
  await nextTick()
  memberList.value?.querySelector<HTMLInputElement>('input')?.focus()
}

function leaveMemberList(event: FocusEvent) {
  if (!memberList.value?.contains(event.relatedTarget as Node | null)) choosingMembers.value = false
}

function exitMemberList() {
  choosingMembers.value = false
  memberListEntry.value?.focus()
}

function focusCollector() {
  document.getElementById('new-member-spelling')?.focus()
}

function selectEntry(member: Entry) {
  if (member.id !== props.entry?.id && !selected.value.some(value => value.id === member.id)) {
    selected.value.push(member)
  }
  addingEntry.value = false
  inlineDraft.value = false
  void search()
}

function resetDraft() {
  selected.value = props.group ? [...props.group.members] : []
  type.value = initialType.value
  note.value = props.group?.note || ''
  collectionDraft.value = false
  inlineDraft.value = false
}
function cancelForm() {
  if (dirty.value && !confirmDiscard()) return
  resetDraft()
  emit('cancel')
}

async function remove() {
  if (!props.group || busy.value || !window.confirm('确定删除整个关系组吗？所有成员词条都会保留。')) return
  error.value = ''
  existingGroup.value = undefined
  deleting.value = true
  try {
    await deleteGroup(props.group.id, props.group.etag)
    resetDraft()
    deleting.value = false
    emit('deleted')
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    deleting.value = false
  }
}

async function save() {
  if (addingEntry.value || collectionDraft.value || (!props.group && memberCount.value < 2)) return
  error.value = ''
  existingGroup.value = undefined
  saving.value = true
  try {
    if (props.group && selected.value.length < 2) {
      if (window.confirm('剩余成员不足两个，保存后将删除整个关系组；所有词条都会保留。确定继续吗？')) {
        await deleteGroup(props.group.id, props.group.etag)
        resetDraft()
        saving.value = false
        emit('deleted')
      }
      return
    }
    const ids = selected.value.map(member => member.id)
    const group = props.group
      ? await updateGroup(props.group.id, type.value, note.value, ids, props.group.etag)
      : await createGroup(props.entry?.id ?? ids[0]!, type.value, note.value, ids)
    resetDraft()
    saving.value = false
    emit('saved', group)
  } catch (err) {
    error.value = errorMessage(err)
    if (err instanceof ApiError) existingGroup.value = err.body.existingGroup
  } finally {
    saving.value = false
  }
}

onMounted(loadTypes)
watch(query, search, { immediate: true })
</script>

<template>
  <form class="form-card group-form" :class="{ 'collection-form': collecting }" :aria-label="collecting ? '收录词条与建立关系组' : group ? '编辑关系组' : '建立关系组'" @submit.prevent="save">
    <EntryCollector v-if="collecting" :disabled="busy" @selected="selectEntry" @busy="entrySaving = $event" @draft="collectionDraft = $event" />
    <div class="group-preparation" :class="{ 'group-edit-workspace': group }">
      <component :is="standalone ? 'h1' : 'h2'" :id="collecting ? 'create-group-heading' : undefined">{{ group ? '编辑关系组' : '建立关系组' }}</component>
      <p class="subtitle">选择已有词条，一起记录含义、用法或区别。</p>
      <div class="group-members-task">
      <fieldset :disabled="busy">
        <legend>组成员（已选 {{ memberCount }} 个）</legend>
        <p v-if="!memberCount" class="selected-members-empty">尚未选择成员</p>
        <ul v-else class="selected-members">
          <li v-if="entry"><span lang="en">{{ entry.spelling }}</span><small>当前词条</small></li>
          <li v-for="member in selected" :key="member.id">
            <span lang="en">{{ member.spelling }}</span>
            <button type="button" class="text-button" :aria-label="'取消选择 ' + member.spelling" @click="toggle(member)">×</button>
          </li>
        </ul>
        <div class="field">
          <label for="member-search">搜索并多选词条</label>
          <input id="member-search" v-model="query" type="search" placeholder="搜索拼写或释义" autocomplete="off" />
          <p class="field-hint">{{ group ? '切换搜索会保留已选成员。至少保留两个成员。' : entry ? '当前词条始终在组内；切换搜索会保留已选成员。至少选择一个其他词条。' : '切换搜索会保留已选成员。至少选择两个词条。' }}</p>
        </div>
        <div v-if="searchError" class="notice error" role="alert">
          {{ searchError }} <button type="button" class="text-button" @click="search">重试搜索</button>
        </div>
        <p v-else-if="loading" role="status">正在搜索词条…</p>
        <p v-else-if="!candidates.length" class="field-hint">{{ entry ? '没有其他匹配的词条。' : '没有匹配的词条，可以新增缺少的词条。' }}</p>
        <div v-else ref="memberList" @focusout="leaveMemberList" @keydown.esc.stop.prevent="exitMemberList">
          <ul ref="memberListEntry" class="member-options" aria-label="可选词条" tabindex="0" @keydown.enter.self.prevent="enterMemberList" @keydown.space.self.prevent="enterMemberList">
            <li v-for="member in candidates" :key="member.id">
              <label>
                <input type="checkbox" :tabindex="choosingMembers ? 0 : -1" :checked="selected.some(value => value.id === member.id)" @change="toggle(member)" @keydown.enter.prevent="toggle(member)" />
                <span><strong lang="en">{{ member.spelling }}</strong><small>{{ member.definition || '暂未填写释义' }}</small></span>
              </label>
            </li>
          </ul>
        </div>
        <button v-if="collecting" type="button" class="button secondary" @click="focusCollector">收录缺少的词条</button>
        <template v-else>
          <button v-if="!addingEntry" type="button" class="button secondary" @click="addingEntry = true">＋ 新增缺少的词条</button>
          <EntryCollector v-else inline :initial-spelling="query" :disabled="busy" @selected="selectEntry" @busy="entrySaving = $event" @draft="inlineDraft = $event" @cancel="addingEntry = false; inlineDraft = false" />
        </template>
      </fieldset>
      </div>
      <div class="group-relation-task">
      <fieldset :disabled="busy">
        <div class="field relation-fields">
          <label for="group-type">关系类型 <span class="required">必填</span></label>
          <div v-if="typesError" class="notice error" role="alert">
            {{ typesError }} <button type="button" class="text-button" @click="loadTypes">重试类型</button>
          </div>
          <select id="group-type" v-model="type" required>
            <option disabled value="">请选择一种关系类型</option>
            <option v-for="kind in types" :key="kind" :value="kind">{{ kind }}</option>
          </select>
        </div>
        <div class="field">
          <label for="group-note">关系说明 <span class="optional">选填</span></label>
          <textarea id="group-note" v-model="note" rows="5" placeholder="写下这一组词的含义、用法或区别…"></textarea>
          <p class="field-hint">支持多行纯文本；所有成员共享这一份说明。</p>
        </div>
      </fieldset>
      <div v-if="error" class="notice error" role="alert">
        <p>{{ error }}</p>
        <RouterLink v-if="existingGroup" :to="'/groups/' + existingGroup.id">
          编辑已有组：{{ existingGroup.type }} · {{ existingGroup.members.map(member => member.spelling).join(' / ') }} →
        </RouterLink>
      </div>
      <div class="form-actions group-actions">
        <button v-if="group" class="button secondary delete-group" type="button" :disabled="busy" @click="remove">{{ deleting ? '正在删除…' : '删除关系组' }}</button>
        <button class="button secondary" type="button" :disabled="busy" @click="cancelForm">{{ group ? '取消编辑' : '取消建组' }}</button>
        <span v-if="saveHint && !busy" class="field-hint save-hint">{{ saveHint }}</span>
        <button class="button primary" type="submit" :disabled="busy || !!saveHint">
          {{ saving ? '正在保存…' : '保存关系组' }}
        </button>
      </div>
      </div>
    </div>
  </form>
</template>
