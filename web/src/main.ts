import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import EntryList from './pages/EntryList.vue'
import EntryForm from './pages/EntryForm.vue'
import EntryDetail from './pages/EntryDetail.vue'
import './style.css'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: EntryList },
    { path: '/entries/new', component: EntryForm },
    { path: '/entries/:id', component: EntryDetail, props: true },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

createApp(App).use(router).mount('#app')
