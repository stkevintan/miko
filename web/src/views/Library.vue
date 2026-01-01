<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '../api';
import Breadcrumb from 'primevue/breadcrumb';
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import Button from 'primevue/button';
import Card from 'primevue/card';
import Panel from 'primevue/panel';
import Divider from 'primevue/divider';
import Dialog from 'primevue/dialog';
import InputText from 'primevue/inputtext';
import InputNumber from 'primevue/inputnumber';
import { useAuthStore } from '../stores/auth';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

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
const loading = ref(false);
const folders = ref<Folder[]>([]);
const currentDir = ref<Directory | null>(null);
const breadcrumbs = ref<any[]>([]);
const selectedSong = ref<any>(null);
const selectedItems = ref<any[]>([]);
const isSelectionMode = ref(false);
const coverArtError = ref(false);
const coverArtLoaded = ref(false);
const editDialogVisible = ref(false);
const editingItem = ref<{id: string, path: string, coverArt?: string} | null>(null);
const editingTags = ref<Array<{key: string, value: string}>>([]);
const coverInput = ref<HTMLInputElement | null>(null);
const coverFile = ref<File | null>(null);
const coverPreview = ref<string | null>(null);
const saving = ref(false);
const refreshKey = ref(Date.now());

watch(selectedSong, () => {
  coverArtError.value = false;
  coverArtLoaded.value = false;
});

const selectionValue = computed({
  get: () => isSelectionMode.value ? selectedItems.value : selectedSong.value,
  set: (val) => {
    if (isSelectionMode.value) {
      selectedItems.value = val as any[];
    } else {
      selectedSong.value = val;
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

interface BreadCrumbItem {
  label: string;
  command?: () => void | Promise<void>;
}
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

const formatSize = (bytes: number) => {
  if (!bytes) return '-';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const formatDuration = (seconds: number) => {
  if (!seconds) return '-';
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
};

const getCoverArtUrl = (id: string) => {
  if (!id) return '';
  return `${api.defaults.baseURL}/library/coverArt?id=${id}&token=${authStore.token}&t=${refreshKey.value}`;
};

const editItem = async (item: any) => {
  loading.value = true;
  try {
    const response = await api.get(`/library/song/tags?id=${item.id}`);
    editingItem.value = { ...item };
    editingTags.value = Object.entries(response.data).flatMap(([key, values]) => 
      (values as string[]).map(value => ({ key, value }))
    );
    coverFile.value = null;
    coverPreview.value = null;
    editDialogVisible.value = true;
  } catch (error) {
    console.error('Failed to fetch tags:', error);
  } finally {
    loading.value = false;
  }
};

const triggerCoverUpload = () => {
  coverInput.value?.click();
};

const onCoverChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  if (target.files && target.files[0]) {
    coverFile.value = target.files[0];
    coverPreview.value = URL.createObjectURL(coverFile.value);
  }
};

const addTag = () => {
  editingTags.value.push({ key: '', value: '' });
};

const removeTag = (index: number) => {
  editingTags.value.splice(index, 1);
};

const saveEdit = async () => {
  if (!editingItem.value) return;
  saving.value = true;
  try {
    const tagsMap: Record<string, string[]> = {};
    editingTags.value.forEach(tag => {
      const key = tag.key.trim();
      const val = tag.value.trim();
      if (key && val) {
        if (!tagsMap[key]) {
          tagsMap[key] = [];
        }
        tagsMap[key].push(val);
      }
    });

    const response = await api.post('/library/song/update', {
      id: editingItem.value.id,
      tags: tagsMap
    });

    // Upload cover if changed
    if (coverFile.value) {
      const formData = new FormData();
      formData.append('id', editingItem.value.id);
      formData.append('file', coverFile.value);
      await api.post('/library/song/cover', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });
      refreshKey.value = Date.now();
    }
    
    // Update local state
    if (currentDir.value) {
      const index = currentDir.value.child.findIndex(c => c.id === editingItem.value?.id);
      if (index !== -1) {
        currentDir.value.child[index] = response.data;
      }
    }
    if (selectedSong.value?.id === editingItem.value.id) {
      selectedSong.value = response.data;
    }
    
    editDialogVisible.value = false;
  } catch (error) {
    console.error('Failed to save edit:', error);
  } finally {
    saving.value = false;
  }
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

        <div class="flex-1 flex flex-col min-w-0 border border-surface-200 dark:border-surface-800 rounded-lg bg-surface-0 dark:bg-surface-900 overflow-hidden">
          <DataTable 
            :value="currentDir?.child" 
            :loading="loading" 
            scrollable
            scrollHeight="flex"
            resizableColumns
            class="p-datatable-sm flex-1"
            v-model:selection="selectionValue"
            :selectionMode="isSelectionMode ? 'multiple' : 'single'"
            dataKey="id"
            @row-click="onRowClick"
            @row-dblclick="(e) => navigate(e.data)"
            paginator
            :rows="50"
            :rowsPerPageOptions="[20, 50, 100]"
            paginatorTemplate="FirstPageLink PrevPageLink PageLinks NextPageLink LastPageLink CurrentPageReport RowsPerPageDropdown"
            currentPageReportTemplate="{first} to {last} of {totalRecords}"
          >
            <Column v-if="isSelectionMode" selectionMode="multiple" headerStyle="width: 3rem"></Column>
            <Column field="title" header="Name" headerStyle="padding-left: 1rem" bodyStyle="padding-left: 1rem">
            <template #body="slotProps">
              <div class="flex items-center max-w-90">
                <i :class="slotProps.data.isDir ? 'pi pi-folder mr-2 text-yellow-500' : 'pi pi-file mr-2 text-blue-500'"></i>
                <span class="truncate">{{ slotProps.data.title }}</span>
              </div>
            </template>
          </Column>
          <Column field="artist" header="Artist" class="hidden lg:table-cell truncate max-w-90"></Column>
          <Column field="album" header="Album" class="hidden xl:table-cell truncate max-w-90"></Column>
          <Column field="duration" header="Duration" class="hidden sm:table-cell w-24">
            <template #body="slotProps">
              {{ formatDuration(slotProps.data.duration) }}
            </template>
          </Column>
          <Column field="bitRate" header="Bitrate" class="hidden md:table-cell w-24">
            <template #body="slotProps">
              <span class="text-nowrap">
              {{ slotProps.data.bitRate ? slotProps.data.bitRate + ' kbps' : '-' }}
              </span>
            </template>
          </Column>
          <Column header="Actions" style="width: 9rem" frozen alignFrozen="right">
            <template #body="slotProps">
              <div class="flex gap-1 justify-center">
                <Button icon="pi pi-pencil" variant="text" severity="secondary" rounded size="small" @click.stop="editItem(slotProps.data)" v-tooltip="'Edit'" />
                <Button icon="pi pi-search" variant="text" severity="secondary" rounded size="small" @click.stop="scrapeItem(slotProps.data)" v-tooltip="'Scrape'" />
                <Button icon="pi pi-trash" variant="text" severity="danger" rounded size="small" @click.stop="deleteItem(slotProps.data)" v-tooltip="'Delete'" />
              </div>
            </template>
          </Column>
          </DataTable>
        </div>
      </div>

      <!-- Right: Detailed View -->
      <div class="w-80 shrink-0 flex flex-col gap-4 overflow-y-auto">
        <Panel v-if="selectedSong" :header="selectedSong.isDir ? 'Directory Details' : 'Song Details'" class="h-full">
          <div v-if="!selectedSong.isDir" class="flex flex-col gap-4">
            <div class="w-full aspect-square bg-surface-100 dark:bg-surface-800 rounded-lg overflow-hidden flex items-center justify-center border border-surface-200 dark:border-surface-700 relative">
               <i class="pi pi-headphones text-6xl text-surface-400"></i>
               <img 
                 v-if="selectedSong.coverArt && !coverArtError" 
                 v-show="coverArtLoaded"
                 :src="getCoverArtUrl(selectedSong.coverArt)" 
                 class="absolute inset-0 w-full h-full object-cover" 
                 alt="Cover Art" 
                 @load="coverArtLoaded = true"
                 @error="coverArtError = true"
               />
            </div>
            <div class="grid grid-cols-1 gap-3 text-sm">
              <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Title</div><div class="text-base font-semibold">{{ selectedSong.title }}</div></div>
              <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Artist</div><div>{{ selectedSong.artist || '-' }}</div></div>
              <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Album</div><div>{{ selectedSong.album || '-' }}</div></div>
              <div class="grid grid-cols-2 gap-2">
                <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Track</div><div>{{ selectedSong.track || '-' }}</div></div>
                <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Year</div><div>{{ selectedSong.year || '-' }}</div></div>
              </div>
              <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Genre</div><div>{{ selectedSong.genre || '-' }}</div></div>
              <div class="grid grid-cols-2 gap-2">
                <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Bitrate</div><div>{{ selectedSong.bitRate }} kbps</div></div>
                <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Format</div><div>{{ selectedSong.suffix }}</div></div>
              </div>
              <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Size</div><div>{{ formatSize(selectedSong.size) }}</div></div>
              <div><div class="font-bold text-surface-500 uppercase text-xs mb-1">Path</div><div class="break-all text-xs opacity-70">{{ selectedSong.path }}</div></div>
            </div>
          </div>
          <div v-else class="flex flex-col gap-4">
            <div class="w-full aspect-square bg-surface-100 dark:bg-surface-800 rounded-lg overflow-hidden flex items-center justify-center border border-surface-200 dark:border-surface-700">
               <i class="pi pi-folder text-6xl text-yellow-500"></i>
            </div>
            <div>
              <div class="font-bold text-surface-500 uppercase text-xs mb-1">Name</div>
              <div class="text-base font-semibold">{{ selectedSong.title }}</div>
            </div>
            <Button label="Open Directory" icon="pi pi-folder-open" class="w-full" @click="navigate(selectedSong)" />
          </div>
        </Panel>
        <div v-else class="h-full flex flex-col items-center justify-center border-2 border-dashed border-surface-200 dark:border-surface-800 rounded-lg text-surface-400 p-6 text-center">
          <i class="pi pi-info-circle text-4xl mb-4"></i>
          <p>Select a file or folder to view detailed information</p>
        </div>
      </div>
    </div>

    <Dialog v-model:visible="editDialogVisible" header="Edit Song Metadata" :modal="true" :style="{ width: '600px' }">
      <div v-if="editingItem" class="flex flex-col gap-4 pt-2">
        <div class="flex gap-4 items-start">
          <div class="w-32 h-32 shrink-0 bg-surface-100 dark:bg-surface-800 rounded-lg overflow-hidden flex items-center justify-center border border-surface-200 dark:border-surface-700 relative group cursor-pointer" @click="triggerCoverUpload">
             <i class="pi pi-headphones text-4xl text-surface-400"></i>
             <img 
               v-if="coverPreview || editingItem.coverArt" 
               :src="coverPreview || getCoverArtUrl(editingItem.coverArt!)" 
               class="absolute inset-0 w-full h-full object-cover" 
               alt="Cover Art" 
             />
             <div class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
               <i class="pi pi-camera text-white text-2xl"></i>
             </div>
             <input type="file" ref="coverInput" class="hidden" accept="image/*" @change="onCoverChange" />
          </div>
          <div class="flex-1 flex flex-col gap-1 min-w-0">
            <div class="font-bold text-surface-500 uppercase text-xs mb-1">File Path</div>
            <div class="text-sm text-surface-600 break-all line-clamp-2 mb-2">{{ editingItem.path }}</div>
            <div class="text-xs text-surface-400 italic">Click the image to upload a new cover art</div>
          </div>
        </div>

        <Divider />
        
        <div class="max-h-[40vh] overflow-y-auto pr-2 flex flex-col gap-3">
          <div v-for="(tag, index) in editingTags" :key="index" class="flex gap-2 items-start">
            <div class="flex-1 flex flex-col gap-1">
              <InputText v-model="tag.key" placeholder="Tag Key (e.g. ARTIST)" class="w-full font-mono text-sm" />
            </div>
            <div class="flex-[2] flex flex-col gap-1">
              <InputText v-model="tag.value" placeholder="Tag Value" class="w-full text-sm" />
            </div>
            <Button icon="pi pi-trash" severity="danger" variant="text" rounded @click="removeTag(index)" />
          </div>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-between w-full">
          <Button icon="pi pi-plus" label="Add Tag" text severity="contrast" @click="addTag" class="whitespace-nowrap" />
          <div class="flex gap-2">
            <Button label="Cancel" icon="pi pi-times" text severity="secondary" @click="editDialogVisible = false" :disabled="saving" />
            <Button label="Save" icon="pi pi-check" @click="saveEdit" :loading="saving" />
          </div>
        </div>
      </template>
    </Dialog>
  </div>
</template>
