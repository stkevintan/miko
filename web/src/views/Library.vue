<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '../api';
import Breadcrumb from 'primevue/breadcrumb';
import Button from 'primevue/button';
import Card from 'primevue/card';
import Divider from 'primevue/divider';
import MetadataDialog from '../components/MetadataDialog.vue';
import LibraryDetail from '../components/LibraryDetail.vue';
import LibraryTable from '../components/LibraryTable.vue';

const route = useRoute();
const router = useRouter();

interface Folder {
  id: number;
  name: string;
  path: string;
}

interface Child {
  id: string;
  title: string;
  isDir: boolean;
  [key: string]: any;
}
interface Directory {
  id: string;
  name: string;
  child: Child[]
}

interface BreadCrumbItem {
  label: string;
  command?: () => void | Promise<void>;
}
const loading = ref(false);
const folders = ref<Folder[]>([]);
const currentDir = ref<Directory | null>(null);
const breadcrumbs = ref<BreadCrumbItem[]>([]);
const selectedSong = ref<Child | null>(null);
const selectedItems = ref<Child[]>([]);
const isSelectionMode = ref(false);
const editDialogVisible = ref(false);
const editingItem = ref<Child | null>(null);
const refreshKey = ref(Date.now());

const selectionValue = computed({
  get: () => isSelectionMode.value ? selectedItems.value : selectedSong.value,
  set: (val) => {
    if (isSelectionMode.value) {
      selectedItems.value = val as Child[];
    } else {
      selectedSong.value = val as Child | null;
    }
  }
});

const toggleSelectionMode = () => {
  isSelectionMode.value = !isSelectionMode.value;
  if (!isSelectionMode.value) {
    selectedItems.value = [];
  }
};

const fetchFolders = async () => {
  loading.value = true;
  try {
    const response = await api.get('/library/folders');
    folders.value = response.data;
  } catch (error) {
    console.error('Failed to fetch folders:', error);
  } finally {
    loading.value = false;
  }
};

const fetchDirectory = async (id: string) => {
  loading.value = true;
  try {
    const response = await api.get(`/library/directory?id=${id}`);
    currentDir.value = response.data;
    updateBreadcrumbs();
  } catch (error) {
    console.error('Failed to fetch directory:', error);
  } finally {
    loading.value = false;
  }
};


const updateBreadcrumbs = () => {
  const items: BreadCrumbItem[] = [{ label: 'Library', command: () => void router.push('/library') }];
  
  if (currentDir.value && route.query.id) {
    items.push({ label: currentDir.value.name });
  }

  breadcrumbs.value = items;
};

const navigate = (item: any) => {
  if (item.isDir) {
    selectedSong.value = null;
    selectedItems.value = [];
    router.push({ 
      path: '/library', 
      query: { 
        id: item.id 
      } 
    });
  }
};

const onRowClick = (event: any) => {
  selectedSong.value = event.data;
};

const selectFolder = (folder: any) => {
  if (folder.directoryId) {
    router.push({ path: '/library', query: { id: folder.directoryId } });
  }
};

onMounted(async () => {
  await fetchFolders();
  handleRoute();
});

const handleRoute = () => {
  const { id } = route.query;
  selectedItems.value = [];
  if (id) {
    fetchDirectory(id as string);
  } else {
    currentDir.value = null;
    updateBreadcrumbs();
  }
};

watch(() => route.query, () => {
  handleRoute();
});

const editItem = (item: any) => {
  editingItem.value = item;
  editDialogVisible.value = true;
};

const onMetadataSave = (updatedItem: any) => {
  // Update local state
  if (currentDir.value) {
    const index = currentDir.value.child.findIndex(c => c.id === updatedItem.id);
    if (index !== -1) {
      currentDir.value.child[index] = updatedItem;
    }
  }
  if (selectedSong.value?.id === updatedItem.id) {
    selectedSong.value = updatedItem;
  }
  refreshKey.value = Date.now();
};

const scrapeItem = (item: any) => {
  console.log('Scrape item:', item);
  // TODO: Implement scrape logic
};

const deleteItem = (item: any) => {
  console.log('Delete item:', item);
  // TODO: Implement delete logic
};

const batchScrape = () => {
  console.log('Batch scrape:', selectedItems.value);
};

const batchDelete = () => {
  console.log('Batch delete:', selectedItems.value);
};
</script>

<template>
  <div class="p-4 h-full flex flex-col overflow-hidden">
    <Breadcrumb :model="breadcrumbs" class="mb-4 shrink-0" />

    <div v-if="!route.query.id" class="flex-1 overflow-y-auto">
      <h2 class="text-2xl font-bold mb-4">Music Folders</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card v-for="folder in folders" :key="folder.id" class="cursor-pointer hover:shadow-lg transition-shadow" @click="selectFolder(folder)">
          <template #title>{{ folder.name }}</template>
          <template #content>
            <p class="text-sm text-gray-500">{{ folder.path }}</p>
          </template>
        </Card>
      </div>
    </div>

    <div v-else class="flex flex-1 gap-4 overflow-hidden">
      <!-- Left: File List -->
      <div class="flex-1 flex flex-col min-w-0 gap-4 overflow-hidden">
        <!-- Action Bar -->
        <div class="flex items-center justify-between p-2 bg-surface-100 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-lg shrink-0">
          <div class="flex items-center gap-2 px-2">
            <Button 
              :icon="isSelectionMode ? 'pi pi-check-square' : 'pi pi-list'" 
              :label="isSelectionMode ? 'Selection Mode' : 'Browse Mode'" 
              size="small" 
              variant="text"
              :severity="isSelectionMode ? 'primary' : 'secondary'"
              @click="toggleSelectionMode" 
            />
            <template v-if="isSelectionMode && selectedItems.length > 0">
              <Divider layout="vertical" class="mx-0 h-4" />
              <span class="text-sm font-medium">{{ selectedItems.length }} selected</span>
              <Button icon="pi pi-times" variant="text" severity="secondary" rounded size="small" @click="selectedItems = []" />
            </template>
          </div>
          <div v-if="isSelectionMode && selectedItems.length > 0" class="flex gap-2">
            <!-- <Button label="Edit" icon="pi pi-pencil" size="small" severity="secondary" @click="batchEdit" /> -->
            <Button label="Scrape" icon="pi pi-search" size="small" severity="secondary" @click="batchScrape" />
            <Button label="Delete" icon="pi pi-trash" size="small" severity="danger" @click="batchDelete" />
          </div>
        </div>

        <LibraryTable 
          :items="currentDir?.child || []" 
          :loading="loading" 
          :selectionMode="isSelectionMode ? 'multiple' : 'single'" 
          v-model:selection="selectionValue" 
          @row-click="onRowClick" 
          @row-dblclick="(e) => navigate(e.data)" 
          @edit="editItem" 
          @scrape="scrapeItem" 
          @delete="deleteItem" 
        />
      </div>

      <!-- Right: Detailed View -->
      <LibraryDetail 
        :item="selectedSong" 
        :refreshKey="refreshKey" 
        @navigate="navigate" 
      />
    </div>

    <MetadataDialog 
      v-model:visible="editDialogVisible" 
      :item="editingItem" 
      @save="onMetadataSave" 
    />
  </div>
</template>
