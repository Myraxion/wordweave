import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import EntryList from './pages/EntryList.vue'
import CollectionCreate from './pages/CollectionCreate.vue'
import EntryDetail from './pages/EntryDetail.vue'
import GroupEdit from './pages/GroupEdit.vue'
import RelationBrowse from './pages/RelationBrowse.vue'
import './style.css'
import { blockWhileSaving, confirmDiscard, hasUnsavedChanges, isSaving } from './leaveProtection'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: EntryList },
    { path: '/entries/new', component: CollectionCreate },
    { path: '/entries/:id', component: EntryDetail, props: true },
    { path: '/groups/new', component: CollectionCreate },
    { path: '/groups/:id', component: GroupEdit, props: true },
    { path: '/relations', component: RelationBrowse },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
  scrollBehavior: to => to.path === '/groups/new' ? { el: '#create-group-heading', top: 24 } : { top: 0 },
})

router.beforeEach(() => {
  if (blockWhileSaving()) return false
  return !hasUnsavedChanges() || confirmDiscard()
})
window.addEventListener('beforeunload', event => {
  if (hasUnsavedChanges() || isSaving()) { event.preventDefault(); event.returnValue = '' }
})

createApp(App).use(router).mount('#app')
