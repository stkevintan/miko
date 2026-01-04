<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { useToast } from 'primevue/usetoast';
import Button from 'primevue/button';
import Menu from 'primevue/menu';
import Avatar from 'primevue/avatar';
import Popover from 'primevue/popover';
import Badge from 'primevue/badge';

const authStore = useAuthStore();
const router = useRouter();
const toast = useToast();

const menuItems = computed(() => [
  {
    label: 'Home',
    icon: 'pi pi-home',
    command: () => router.push('/')
  },
  {
    label: 'Library',
    icon: 'pi pi-list',
    badge: scanCount.value > 0 ? scanCount.value.toString() : undefined,
    command: () => router.push('/library')
  },
  {
    label: 'Search',
    icon: 'pi pi-search',
    command: () => router.push('/search')
  }
]);

const op = ref();
const toggleAccountMenu = (event: Event) => {
  op.value.toggle(event);
};

const handleLogout = () => {
  authStore.logout();
  router.push('/login');
};

import { useTheme } from '../composables/useTheme';
import api from '../api';
import { onMounted, onUnmounted } from 'vue';

const { isDark, toggleDarkMode } = useTheme();

const isScanning = ref(false);
const isScraping = ref(false);
const scanCount = ref(0);

const checkStatus = async () => {
  try {
    const response = await api.get('/library/status');
    isScanning.value = response.data.scanning;
    isScraping.value = response.data.scraping;
    scanCount.value = response.data.count;
  } catch (error) {
    console.error('Failed to fetch status:', error);
    // Silent fail for status check to avoid spamming UI
  }
};

let statusInterval: number;

onMounted(() => {
  checkStatus();
  statusInterval = window.setInterval(checkStatus, 5000);
});

onUnmounted(() => {
  if (statusInterval) window.clearInterval(statusInterval);
});

const scanMenu = ref();
const scanMenuItems = ref([
  {
    label: 'Incremental Scan',
    icon: 'pi pi-refresh',
    command: () => startScan(true)
  },
  {
    label: 'Full Scan',
    icon: 'pi pi-sync',
    command: () => startScan(false)
  }
]);

const toggleScanMenu = (event: Event) => {
  scanMenu.value.toggle(event);
};

const scrapeMenu = ref();
const scrapeMenuItems = ref([
  {
    label: 'Incremental Scrape',
    icon: 'pi pi-search-plus',
    command: () => startScrape('inc')
  },
  {
    label: 'Full Scrape',
    icon: 'pi pi-search',
    command: () => startScrape('full')
  }
]);

const toggleScrapeMenu = (event: Event) => {
  scrapeMenu.value.toggle(event);
};

const startScan = async (incremental: boolean) => {
  isScanning.value = true;
  try {
    await api.post(`/library/scan/all?incremental=${incremental}`);
    toast.add({ 
      severity: 'success', 
      summary: 'Success', 
      detail: `${incremental ? 'Incremental' : 'Full'} library scan started`, 
      life: 3000 
    });
  } catch (error: any) {
    toast.add({ 
      severity: 'error', 
      summary: 'Scan Failed', 
      detail: error.response?.data?.error || error.message || 'Failed to start scan', 
      life: 5000 
    });
    isScanning.value = false;
  }
};

const startScrape = async (mode: string) => {
  isScraping.value = true;
  try {
    await api.post(`/library/song/scrape/all?mode=${mode}`);
    toast.add({ 
      severity: 'success', 
      summary: 'Success', 
      detail: `${mode === 'inc' ? 'Incremental' : 'Full'} library scrape started`, 
      life: 3000 
    });
  } catch (error: any) {
    toast.add({ 
      severity: 'error', 
      summary: 'Scrape Failed', 
      detail: error.response?.data?.error || error.message || 'Failed to start scrape', 
      life: 5000 
    });
    isScraping.value = false;
  }
};
</script>

<template>
  <div class="min-h-screen h-full flex flex-col bg-surface-50 dark:bg-surface-950">
    <!-- Topbar -->
    <header class="h-16 flex items-center justify-between px-6 bg-surface-0 dark:bg-surface-900 border-b border-surface-200 dark:border-surface-800 sticky top-0 z-50">
      <div class="flex items-center gap-2">
        <i class="pi pi-box text-primary text-2xl"></i>
        <span class="text-xl font-bold text-surface-900 dark:text-surface-0">Miko</span>
      </div>
      
      <div class="flex items-center gap-3">
        <Button 
          :icon="isScanning ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" 
          v-tooltip="isScanning ? 'Scanning music folders...' : 'Scan options'"
          variant="text" 
          severity="secondary" 
          size="small"
          :disabled="isScanning"
          @click="toggleScanMenu"
        />
        <Menu ref="scanMenu" :model="scanMenuItems" :popup="true" />
        <Button 
          :icon="isScraping ? 'pi pi-spin pi-spinner' : 'pi pi-search-plus'" 
          v-tooltip="isScraping ? 'Scraping metadata...' : 'Scrape options'"
          variant="text" 
          severity="secondary" 
          size="small"
          :disabled="isScraping"
          @click="toggleScrapeMenu"
        />
        <Menu ref="scrapeMenu" :model="scrapeMenuItems" :popup="true" />
        <Button 
          :icon="isDark ? 'pi pi-sun' : 'pi pi-moon'" 
          variant="text" 
          severity="secondary" 
          rounded 
          @click="toggleDarkMode"
        />
        <Button icon="pi pi-cog" variant="text" severity="secondary" rounded />
        <Button 
          @click="toggleAccountMenu" 
          variant="text" 
          severity="secondary" 
          class="p-0 overflow-hidden rounded-full"
        >
          <Avatar icon="pi pi-user" shape="circle" />
        </Button>
        
        <Popover ref="op">
          <div class="flex flex-col gap-2 w-48">
            <div class="px-3 py-2 border-b border-surface-200 dark:border-surface-800">
              <p class="font-semibold m-0">{{ authStore.user?.username || 'User' }}</p>
              <p class="text-sm text-surface-500 m-0">Account Settings</p>
            </div>
            <div class="flex flex-col">
              <Button label="Profile" icon="pi pi-user" variant="text" severity="secondary" class="justify-start" @click="router.push('/profile')" />
              <Button label="Settings" icon="pi pi-cog" variant="text" severity="secondary" class="justify-start" />
              <Button label="Logout" icon="pi pi-sign-out" variant="text" severity="danger" class="justify-start" @click="handleLogout" />
            </div>
          </div>
        </Popover>
      </div>
    </header>

    <div class="flex flex-1 overflow-hidden">
      <!-- Sidebar -->
      <aside class="w-64 hidden md:flex flex-col bg-surface-0 dark:bg-surface-900 border-r border-surface-200 dark:border-surface-800">
        <nav class="flex-1 p-4">
          <Menu :model="menuItems" class="w-full border-none bg-transparent">
            <template #item="{ item, props }">
              <a class="flex items-center px-4 py-3 cursor-pointer group" v-bind="props.action">
                <span :class="[item.icon, 'text-surface-600 dark:text-surface-400 group-hover:text-primary']" />
                <span class="ml-2 font-medium text-surface-700 dark:text-surface-0 group-hover:text-primary">{{ item.label }}</span>
                <Badge v-if="item.badge" class="ml-auto" :value="item.badge" severity="secondary" />
              </a>
            </template>
          </Menu>
        </nav>
      </aside>

      <!-- Main Content -->
      <main class="flex-1 overflow-y-auto p-6">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
:deep(.p-menu) {
  background: transparent;
  border: none;
}

:deep(.p-menuitem-link) {
  border-radius: 8px;
  margin-bottom: 4px;
}
</style>
