import { ref, watch } from 'vue'

export type BrowseLayout = 'grid' | 'masonry'

export function useBrowseLayout(storageKey: string) {
  let saved: BrowseLayout = 'grid'
  try {
    if (localStorage.getItem(storageKey) === 'masonry') saved = 'masonry'
  } catch {
    // Browsing still works when browser storage is unavailable.
  }
  const layout = ref<BrowseLayout>(saved)
  watch(layout, value => {
    try {
      localStorage.setItem(storageKey, value)
    } catch {
      // Layout switching does not require browser storage.
    }
  })
  return layout
}
