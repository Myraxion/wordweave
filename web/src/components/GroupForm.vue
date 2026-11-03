<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ApiError, createGroup, errorMessage, getGroupTypes, listEntries, type Entry, type Group } from '../api'

const props = defineProps<{ entry: Entry }>()
const emit = defineEmits<{ saved: [group: Group]; cancel: [] }>()
const types = ref<string[]>([])
const type = ref('')
const note = ref('')
const query = ref('')
const results = ref<Entry[]>([])
const selected = ref<Entry[]>([])
const loading = ref(false)
const searchError = ref('')
const typesError = ref('')
const saving = ref(false)
const error = ref('')
const existingGroup = ref<Group>()
const candidates = computed(() => results.value.filter(member => member.id !== props.entry.id))
let searchNumber = 0

async function loadTypes() {
  typesError.value = ''
  try {
    types.value = await getGroupTypes()
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

async function save() {
  error.value = ''
  existingGroup.value = undefined
  saving.value = true
  try {
    const group = await createGroup(props.entry.id, type.value, note.value, selected.value.map(member => member.id))
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
  <form class="form-card group-form" aria-label="建立关系组" @submit.prevent="save">
    <h2>建立关系组</h2>
    <p class="subtitle">选择已有词条，一起记录含义、用法或区别。</p>
    <fieldset :disabled="saving">
      <legend>组成员（已选 {{ selected.length + 1 }} 个）</legend>
      <ul class="selected-members">
        <li><span lang="en">{{ entry.spelling }}</span><small>当前词条</small></li>
        <li v-for="member in selected" :key="member.id">
          <span lang="en">{{ member.spelling }}</span>
          <button type="button" class="text-button" :aria-label="'取消选择 ' + member.spelling" @click="toggle(member)">×</button>
        </li>
      </ul>
      <div class="field">
        <label for="member-search">搜索并多选词条</label>
        <input id="member-search" v-model="query" type="search" placeholder="搜索拼写或释义" autocomplete="off" />
        <p class="field-hint">当前词条始终在组内；切换搜索会保留已选成员。至少选择一个其他词条。</p>
      </div>
      <div v-if="searchError" class="notice error" role="alert">
        {{ searchError }} <button type="button" class="text-button" @click="search">重试搜索</button>
      </div>
      <p v-else-if="loading" role="status">正在搜索词条…</p>
      <p v-else-if="!candidates.length" class="field-hint">没有其他匹配的词条。</p>
      <ul v-else class="member-options" aria-label="可选词条">
        <li v-for="member in candidates" :key="member.id">
          <label>
            <input type="checkbox" :checked="selected.some(value => value.id === member.id)" @change="toggle(member)" />
            <span><strong lang="en">{{ member.spelling }}</strong><small>{{ member.definition || '暂未填写释义' }}</small></span>
          </label>
        </li>
      </ul>
      <div class="field">
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
        查看已有组：{{ existingGroup.type }} · {{ existingGroup.members.map(member => member.spelling).join(' / ') }} →
      </RouterLink>
    </div>
    <div class="form-actions">
      <button class="button secondary" type="button" :disabled="saving" @click="emit('cancel')">取消建组</button>
      <button class="button primary" type="submit" :disabled="saving || !type || selected.length === 0">
        {{ saving ? '正在保存…' : '保存关系组' }}
      </button>
    </div>
  </form>
</template>
