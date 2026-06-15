import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'

import App from './App.vue'
import Queries from './pages/Queries.vue'
import Articles from './pages/Articles.vue'
import ArticleDetail from './pages/ArticleDetail.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/queries' },
    { path: '/queries', component: Queries },
    { path: '/articles', component: Articles },
    { path: '/articles/:id', component: ArticleDetail },
  ],
})

createApp(App).use(router).use(ElementPlus).mount('#app')
