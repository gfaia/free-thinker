<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listArticles } from '../api/articles'
import type { Article } from '../types'

const router = useRouter()
const loading = ref(false)
const error = ref('')
const articles = ref<Article[]>([])
const total = ref(0)
const filters = reactive({
  source: '',
  query: '',
  limit: 50,
  offset: 0,
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams()
    if (filters.source) params.set('source', filters.source)
    if (filters.query) params.set('query', filters.query)
    params.set('limit', String(filters.limit))
    params.set('offset', String(filters.offset))
    const data = await listArticles(params)
    articles.value = data.items
    total.value = data.total ?? data.items.length
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

function open(row: Article) {
  router.push(`/articles/${row.id}`)
}

onMounted(load)
</script>

<template>
  <el-card class="page-card">
    <template #header>
      <div class="card-header">
        <div>
          <div class="card-title">Articles</div>
          <div class="card-subtitle">Browse collected articles and open their stored details.</div>
        </div>
        <el-button type="primary" :loading="loading" @click="load">Refresh</el-button>
      </div>
    </template>

    <el-form :model="filters" class="filters" @submit.prevent>
      <el-form-item label="Source">
        <el-input v-model="filters.source" clearable placeholder="zhihu" @keyup.enter="load" />
      </el-form-item>
      <el-form-item label="Query">
        <el-input v-model="filters.query" clearable placeholder="golang" @keyup.enter="load" />
      </el-form-item>
      <el-form-item label="Limit">
        <el-input-number v-model="filters.limit" :min="1" :max="200" />
      </el-form-item>
      <el-form-item label="Offset">
        <el-input-number v-model="filters.offset" :min="0" />
      </el-form-item>
      <el-form-item class="filter-actions">
        <el-button @click="load">Apply</el-button>
      </el-form-item>
    </el-form>

    <el-alert v-if="error" type="error" :title="error" show-icon class="alert" />
    <p class="muted">Total: {{ total }}</p>

    <div class="table-wrap desktop-table">
      <el-table v-loading="loading" :data="articles" empty-text="No articles found" @row-click="open">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="Title" min-width="280">
          <template #default="scope">
            <div class="title">{{ scope.row.title }}</div>
            <el-link :href="scope.row.url" target="_blank" @click.stop>{{ scope.row.url }}</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="source" label="Source" width="120" />
        <el-table-column prop="query_keyword" label="Query" width="160" />
        <el-table-column prop="author" label="Author" width="160" />
        <el-table-column prop="created_at" label="Created" min-width="220" />
      </el-table>
    </div>

    <div v-loading="loading" class="mobile-list">
      <el-empty v-if="!articles.length" description="No articles found" />
      <article v-for="article in articles" :key="article.id" class="mobile-card">
        <h3>{{ article.title || 'Untitled article' }}</h3>
        <div class="mobile-tags">
          <el-tag size="small">{{ article.source || 'unknown' }}</el-tag>
          <el-tag v-if="article.query_keyword" size="small" type="info">{{ article.query_keyword }}</el-tag>
        </div>
        <dl class="meta-list">
          <div>
            <dt>Author</dt>
            <dd>{{ article.author || '-' }}</dd>
          </div>
          <div>
            <dt>Created</dt>
            <dd>{{ article.created_at || '-' }}</dd>
          </div>
        </dl>
        <el-link class="article-url" :href="article.url" target="_blank" @click.stop>
          {{ article.url }}
        </el-link>
        <div class="mobile-actions">
          <el-button type="primary" plain @click="open(article)">View detail</el-button>
        </div>
      </article>
    </div>
  </el-card>
</template>

<style scoped>
.page-card {
  overflow: hidden;
}

.card-header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: center;
}

.card-title {
  font-size: 1.1rem;
  font-weight: 700;
}

.card-subtitle {
  margin-top: .25rem;
  color: var(--el-text-color-secondary);
  font-size: .9rem;
}

.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 0 16px;
  margin-bottom: 1rem;
}

.alert {
  margin-bottom: 1rem;
}

.muted {
  color: var(--el-text-color-secondary);
}

.table-wrap {
  overflow-x: auto;
}

.title {
  font-weight: 600;
  margin-bottom: .25rem;
}

.mobile-list {
  display: none;
}

.mobile-card {
  padding: 1rem;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-bg-color);
}

.mobile-card + .mobile-card {
  margin-top: .75rem;
}

.mobile-card h3 {
  margin: 0;
  overflow-wrap: anywhere;
  font-size: 1rem;
  line-height: 1.4;
}

.mobile-tags {
  display: flex;
  flex-wrap: wrap;
  gap: .5rem;
  margin-top: .75rem;
}

.meta-list {
  display: grid;
  gap: .75rem;
  margin: .9rem 0;
}

.meta-list div {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.meta-list dt {
  color: var(--el-text-color-secondary);
}

.meta-list dd {
  margin: 0;
  text-align: right;
  overflow-wrap: anywhere;
}

.article-url {
  max-width: 100%;
  overflow-wrap: anywhere;
}

.mobile-actions {
  margin-top: 1rem;
}

@media (max-width: 767px) {
  .card-header {
    align-items: flex-start;
  }

  .filters {
    display: block;
  }

  .filters :deep(.el-form-item) {
    display: block;
    margin-right: 0;
  }

  .filters :deep(.el-form-item__label) {
    justify-content: flex-start;
  }

  .filters :deep(.el-input),
  .filters :deep(.el-input-number),
  .filters :deep(.el-button) {
    width: 100%;
  }

  .filter-actions {
    margin-bottom: 0;
  }

  .desktop-table {
    display: none;
  }

  .mobile-list {
    display: block;
  }

  .mobile-actions :deep(.el-button) {
    width: 100%;
  }
}
</style>
