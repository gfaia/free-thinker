<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const drawerVisible = ref(false)

const navItems = [
  { label: 'Queries', path: '/queries' },
  { label: 'Articles', path: '/articles' },
]

function navigate(path: string) {
  drawerVisible.value = false
  if (route.path !== path) {
    router.push(path)
  }
}
</script>

<template>
  <el-container class="layout">
    <el-header class="header">
      <div class="header-inner">
        <div class="brand" @click="navigate('/queries')">Free Thinker Portal</div>

        <el-menu mode="horizontal" router :default-active="route.path" class="menu">
          <el-menu-item v-for="item in navItems" :key="item.path" :index="item.path">
            {{ item.label }}
          </el-menu-item>
        </el-menu>

        <el-button class="mobile-menu-button" text @click="drawerVisible = true">Menu</el-button>
      </div>
    </el-header>

    <el-main class="main">
      <router-view />
    </el-main>

    <el-drawer v-model="drawerVisible" title="Navigation" direction="rtl" size="78%" class="mobile-drawer">
      <el-menu :default-active="route.path" class="drawer-menu">
        <el-menu-item v-for="item in navItems" :key="item.path" :index="item.path" @click="navigate(item.path)">
          {{ item.label }}
        </el-menu-item>
      </el-menu>
    </el-drawer>
  </el-container>
</template>

<style scoped>
:global(*) {
  box-sizing: border-box;
}

:global(body) {
  margin: 0;
  background: var(--el-bg-color-page);
}

:global(#app) {
  min-height: 100vh;
}

.layout {
  min-height: 100vh;
}

.header {
  position: sticky;
  top: 0;
  z-index: 20;
  height: 64px;
  padding: 0;
  border-bottom: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}

.header-inner {
  display: flex;
  align-items: center;
  width: 100%;
  max-width: 1200px;
  height: 100%;
  margin: 0 auto;
  padding: 0 24px;
}

.brand {
  flex: 0 0 auto;
  margin-right: 2rem;
  font-size: 1.05rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}

.menu {
  flex: 1;
  border-bottom: 0;
}

.mobile-menu-button {
  display: none;
}

.main {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
}

.drawer-menu {
  border-right: 0;
}

@media (max-width: 767px) {
  .header {
    height: 56px;
  }

  .header-inner {
    padding: 0 12px;
  }

  .brand {
    flex: 1;
    min-width: 0;
    margin-right: 1rem;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .menu {
    display: none;
  }

  .mobile-menu-button {
    display: inline-flex;
  }

  .main {
    padding: 12px;
  }
}
</style>
