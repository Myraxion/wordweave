import { ref, watch } from 'vue'

export type StudyMode = 'full' | 'hide-definition' | 'hide-spelling'

export const studyModes: { value: StudyMode; label: string }[] = [
  { value: 'full', label: '完整模式' },
  { value: 'hide-definition', label: '遮挡释义' },
  { value: 'hide-spelling', label: '遮挡拼写' },
]

const storageKey = 'wordweave.study-mode'

function savedMode(): StudyMode {
  try {
    const saved = localStorage.getItem(storageKey)
    return studyModes.find(mode => mode.value === saved)?.value ?? 'full'
  } catch {
    return 'full'
  }
}

export const studyMode = ref<StudyMode>(savedMode())

watch(studyMode, mode => {
  try {
    localStorage.setItem(storageKey, mode)
  } catch {
    // Browsing still works when browser storage is unavailable.
  }
})
