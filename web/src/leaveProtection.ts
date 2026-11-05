import { onBeforeUnmount, watch, type Ref } from 'vue'

const drafts = new Set<() => boolean>()
const saves = new Set<() => boolean>()

export function useLeaveProtection(dirty: Ref<boolean>, busy: Ref<boolean>) {
  const hasDraft = () => dirty.value
  const isSaving = () => busy.value
  drafts.add(hasDraft)
  saves.add(isSaving)
  watch(busy, () => window.dispatchEvent(new Event('save-state-changed')), { flush: 'sync' })
  onBeforeUnmount(() => { drafts.delete(hasDraft); saves.delete(isSaving) })
}

export function confirmDiscard() {
  return window.confirm('有未保存的内容，确定放弃吗？')
}

export function hasUnsavedChanges() {
  return [...drafts].some(check => check())
}

export function isSaving() {
  return [...saves].some(check => check())
}
