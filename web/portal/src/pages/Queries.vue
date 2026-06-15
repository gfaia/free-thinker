<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { listQueries } from '../api/queries'
import type { QueryTask } from '../types'

const loading = ref(false)
const error = ref('')
const queries = ref<QueryTask[]>([])
const filters = reactive({
  platform: '',
  status: '',
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams()
    if (filters.platform) params.set('platform', filters.platform)
    if (filters.status) params.set('status', filters.status)
    const data = await listQueries(params)
    queries.value = data.items
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <el-card class="page-card">
    <template #header>
      <div class="card-header">
        <div>
          <div class="card-title">Query Tasks</div>
          <div class="card-subtitle">Track crawler query execution by platform and status.</div>
        </div>
        <el-button type="primary" :loading="loading" @click="load">Refresh</el-button>
      </div>
    </template>

    <el-form :model="filters" class="filters" @submit.prevent>
      <el-form-item label="Platform">
        <el-input v-model="filters.platform" clearable placeholder="zhihu" @keyup.enter="load" />
      </el-form-item>
      <el-form-item label="Status">
        <el-select v-model="filters.status" clearable placeholder="Any" class="status-select">
          <el-option label="completed" value="completed" />
          <el-option label="failed" value="failed" />
        </el-select>
      </el-form-item>
      <el-form-item class="filter-actions">
        <el-button @click="load">Apply</el-button>
      </el-form-item>
    </el-form>

    <el-alert v-if="error" type="error" :title="error" show-icon class="alert" />

    <div class="table-wrap desktop-table">
      <el-table v-loading="loading" :data="queries" empty-text="No query task records found">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="keyword" label="Keyword" min-width="180" />
        <el-table-column prop="platform" label="Platform" width="140" />
        <el-table-column prop="status" label="Status" width="140" />
        <el-table-column prop="last_run" label="Last Run" min-width="220" />
      </el-table>
    </div>

    <div v-loading="loading" class="mobile-list">
      <el-empty v-if="!queries.length" description="No query task records found" />
      <article v-for="query in queries" :key="query.id" class="mobile-card">
        <div class="mobile-card-header">
          <h3>{{ query.keyword }}</h3>
          <el-tag size="small" :type="query.status === 'failed' ? 'danger' : 'success'">
            {{ query.status || 'unknown' }}
          </el-tag>
        </div>
        <dl class="meta-list">
          <div>
            <dt>Platform</dt>
            <dd>{{ query.platform || '-' }}</dd>
          </div>
          <div>
            <dt>Last Run</dt>
            <dd>{{ query.last_run || '-' }}</dd>
          </div>
          <div>
            <dt>ID</dt>
            <dd>#{{ query.id }}</dd>
          </div>
        </dl>
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

.status-select {
  width: 160px;
}

.alert {
  margin-bottom: 1rem;
}

.table-wrap {
  overflow-x: auto;
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

.mobile-card-header {
  display: flex;
  justify-content: space-between;
  gap: .75rem;
  align-items: flex-start;
}

.mobile-card h3 {
  margin: 0;
  overflow-wrap: anywhere;
  font-size: 1rem;
}

.meta-list {
  display: grid;
  gap: .75rem;
  margin: .9rem 0 0;
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
  .filters :deep(.el-select),
  .filters :deep(.el-button) {
    width: 100%;
  }

  .status-select {
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
}
</style>
