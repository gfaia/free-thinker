<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { getArticle, getArticleContent } from '../api/articles'
import type { Article } from '../types'

const route = useRoute()
const article = ref<Article | null>(null)
const rawContent = ref('')
const loading = ref(false)
const contentLoading = ref(false)
const error = ref('')
const contentError = ref('')
const isMobile = ref(false)
let mobileQuery: MediaQueryList | undefined

function updateMobile() {
  isMobile.value = mobileQuery?.matches ?? false
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    article.value = await getArticle(String(route.params.id))
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

async function loadContent() {
  contentLoading.value = true
  contentError.value = ''
  try {
    rawContent.value = await getArticleContent(String(route.params.id))
  } catch (err) {
    contentError.value = err instanceof Error ? err.message : String(err)
  } finally {
    contentLoading.value = false
  }
}

onMounted(() => {
  mobileQuery = window.matchMedia('(max-width: 767px)')
  updateMobile()
  mobileQuery.addEventListener('change', updateMobile)
  load()
})

onBeforeUnmount(() => {
  mobileQuery?.removeEventListener('change', updateMobile)
})
</script>

<template>
  <el-card v-loading="loading" class="detail-card">
    <template #header>
      <div class="card-header">
        <span class="detail-title">{{ article?.title || 'Article Detail' }}</span>
        <el-link v-if="article" :href="article.url" target="_blank">Open Original</el-link>
      </div>
    </template>

    <el-alert v-if="error" type="error" :title="error" show-icon class="alert" />
    <template v-if="article">
      <el-descriptions :column="isMobile ? 1 : 2" border>
        <el-descriptions-item label="ID">{{ article.id }}</el-descriptions-item>
        <el-descriptions-item label="Source">{{ article.source }}</el-descriptions-item>
        <el-descriptions-item label="Query">{{ article.query_keyword }}</el-descriptions-item>
        <el-descriptions-item label="Author">{{ article.author }}</el-descriptions-item>
        <el-descriptions-item label="Published">{{ article.published_at }}</el-descriptions-item>
        <el-descriptions-item label="Created">{{ article.created_at }}</el-descriptions-item>
      </el-descriptions>

      <h3>Summary</h3>
      <p class="summary">{{ article.summary || 'No summary.' }}</p>

      <div v-if="article.content_path" class="content-actions">
        <el-button :loading="contentLoading" @click="loadContent">Load Raw Content As Text</el-button>
      </div>
      <el-alert v-if="contentError" type="error" :title="contentError" show-icon class="alert" />
      <pre v-if="rawContent" class="raw">{{ rawContent }}</pre>
    </template>
  </el-card>
</template>

<style scoped>
.detail-card {
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: center;
}

.detail-title {
  min-width: 0;
  overflow-wrap: anywhere;
  font-weight: 700;
}

.alert {
  margin-bottom: 1rem;
}

.summary {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.7;
}

.content-actions {
  margin: 1rem 0;
}

.raw {
  max-width: 100%;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: #f8f8f8;
  padding: 1rem;
  white-space: pre-wrap;
  overflow: auto;
  overflow-wrap: anywhere;
}

@media (max-width: 767px) {
  .card-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .content-actions :deep(.el-button) {
    width: 100%;
  }

  .raw {
    padding: .75rem;
    font-size: .85rem;
  }
}
</style>
