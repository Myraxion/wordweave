import { nextTick, onBeforeUnmount, watch, type Ref } from 'vue'

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

export function blockWhileSaving() {
  if (!isSaving()) return false
  window.dispatchEvent(new Event('save-navigation-blocked'))
  return true
}

// Keep a successful write protected until its draft has been cleared and Vue
// has unmounted any discarded nested drafts. Follow-up navigation happens after
// the protection is released; it cannot turn a successful write into a failure.
export async function protectedWrite<T>(
  busy: Ref<boolean>,
  write: () => Promise<T>,
  clear: (result: T) => void,
  after?: (result: T) => void | Promise<void>,
): Promise<T> {
  busy.value = true
  let result: T
  try {
    result = await write()
    clear(result)
    await nextTick()
  } finally {
    busy.value = false
  }
  await after?.(result)
  return result
}
