<script setup lang="ts">
import { ref, onMounted, watch, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useToast } from "primevue/usetoast";
import { useConfirm } from "primevue/useconfirm";
import api from "../api";
import Breadcrumb from "primevue/breadcrumb";
import Card from "primevue/card";
import Button from "primevue/button";
import MetadataDialog from "../components/MetadataDialog.vue";
import LibraryDetail from "../components/LibraryDetail.vue";
import LibraryTable from "../components/LibraryTable.vue";
import { Child, Directory, Folder } from "@/types/library";
import { DataTableRowClickEvent } from "primevue";

const route = useRoute();
const router = useRouter();
const toast = useToast();
const confirm = useConfirm();

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
const scanningIds = ref<string[]>([]);
const scrapingIds = ref<string[]>([]);

const rows = ref(50);
const first = ref(0);

const selectionValue = computed({
    get: () =>
        isSelectionMode.value ? selectedItems.value : selectedSong.value,
    set: (val) => {
        if (isSelectionMode.value) {
            selectedItems.value = val as Child[];
        } else {
            selectedSong.value = val as Child | null;
        }
    },
});

const fetchFolders = async () => {
    loading.value = true;
    try {
        const response = await api.get("/library/folders");
        folders.value = response.data;
    } catch (error) {
        console.error("Failed to fetch folders:", error);
    } finally {
        loading.value = false;
    }
};

const fetchDirectory = async (id: string, offset = first.value, limit = rows.value) => {
    loading.value = true;
    try {
        const response = await api.get(`/library/directory?id=${id}&offset=${offset}&limit=${limit}`);
        currentDir.value = response.data;
        updateBreadcrumbs();
    } catch (error) {
        console.error("Failed to fetch directory:", error);
    } finally {
        loading.value = false;
    }
};

const onPage = (event: any) => {
    first.value = event.first;
    rows.value = event.rows;
    if (route.query.id) {
        fetchDirectory(route.query.id as string, first.value, rows.value);
    }
};

const updateBreadcrumbs = () => {
    const items: BreadCrumbItem[] = [
        { label: "Library", command: () => void router.push("/library") },
    ];

    if (currentDir.value && route.query.id) {
        if (currentDir.value.parents) {
            currentDir.value.parents.forEach((p) => {
                items.push({
                    label: p.title,
                    command: () =>
                        void router.push({
                            path: "/library",
                            query: { id: p.id },
                        }),
                });
            });
        }
        items.push({ label: currentDir.value.name });
    }

    breadcrumbs.value = items;
};

const navigate = (item: Child) => {
    // ignore selection mode when navigating
    if (isSelectionMode.value) return;
    if (item.isDir) {
        selectedSong.value = null;
        selectedItems.value = [];
        router.push({
            path: "/library",
            query: {
                id: item.id,
            },
        });
    } else {
        selectedSong.value = item;
        selectedItems.value = [];
        editItem(item);
    }
};

const onRowClick = (event: DataTableRowClickEvent<Child>) => {
    selectedSong.value = event.data;
};

const selectFolder = (folder: Folder) => {
    router.push({ path: "/library", query: { id: folder.directoryId } });
};

const scanFolder = async (folder: Folder) => {
    scanningIds.value.push(folder.directoryId);
    try {
        await api.post("/library/scan", { ids: [folder.directoryId] });
        toast.add({ severity: 'success', summary: 'Success', detail: 'Folder scan started', life: 3000 });
    } catch (error: any) {
        console.error("Failed to scan folder:", error);
        toast.add({ 
            severity: 'error', 
            summary: 'Scan Failed', 
            detail: error.response?.data?.error || error.message || 'Failed to scan folder', 
            life: 5000 
        });
    } finally {
        scanningIds.value = scanningIds.value.filter(id => id !== folder.directoryId);
    }
};

const scrapeFolder = async (folder: Folder) => {
    scrapingIds.value.push(folder.directoryId);
    try {
        await api.post("/library/song/scrape", { ids: [folder.directoryId], mode: 'full' });
        toast.add({ severity: 'success', summary: 'Success', detail: 'Folder scrape started', life: 3000 });
    } catch (error: any) {
        console.error("Failed to scrape folder:", error);
        toast.add({ 
            severity: 'error', 
            summary: 'Scrape Failed', 
            detail: error.response?.data?.error || error.message || 'Failed to scrape folder', 
            life: 5000 
        });
    } finally {
        scrapingIds.value = scrapingIds.value.filter(id => id !== folder.directoryId);
    }
};

onMounted(async () => {
    await fetchFolders();
    handleRoute();
});

const handleRoute = () => {
    const { id } = route.query;
    selectedItems.value = [];
    if (typeof id === "string") {
        if (currentDir.value?.id !== id) {
            first.value = 0;
        }
        fetchDirectory(id);
    } else {
        currentDir.value = null;
        updateBreadcrumbs();
    }
};

watch(
    () => route.query,
    () => {
        handleRoute();
    },
);

const editItem = (item: Child) => {
    editingItem.value = item;
    editDialogVisible.value = true;
};

const onMetadataSave = (updatedItem: Child) => {
    // Update local state
    if (currentDir.value) {
        const index = currentDir.value.child.findIndex(
            (c) => c.id === updatedItem.id,
        );
        if (index !== -1) {
            currentDir.value.child[index] = updatedItem;
        }
    }
    if (selectedSong.value?.id === updatedItem.id) {
        selectedSong.value = updatedItem;
    }
    refreshKey.value = Date.now();
    toast.add({ severity: 'success', summary: 'Success', detail: 'Metadata updated successfully', life: 3000 });
};

const handleUpdatedIds = async (updatedIds: string[]) => {
    // Refresh the directory to get updated data if any of the updated IDs are in the current view
    if (currentDir.value && updatedIds.some(id => currentDir.value?.child.some(c => c.id === id))) {
        await fetchDirectory(route.query.id as string || "");
    }
    
    // Update selected song if it was affected
    if (selectedSong.value && updatedIds.includes(selectedSong.value.id)) {
        const res = await api.get(`/library/song?id=${selectedSong.value.id}`);
        selectedSong.value = res.data;
    }

    refreshKey.value = Date.now();
};

const scanItem = async (item: Child) => {
    scanningIds.value.push(item.id);
    try {
        const response = await api.post("/library/scan", { ids: [item.id] });
        await handleUpdatedIds(response.data as string[]);
        toast.add({ severity: 'success', summary: 'Success', detail: 'Item scanned successfully', life: 3000 });
    } catch (error: any) {
        console.error("Failed to scan item:", error);
        toast.add({ 
            severity: 'error', 
            summary: 'Scan Failed', 
            detail: error.response?.data?.error || error.message || 'Failed to scan item', 
            life: 5000 
        });
    } finally {
        scanningIds.value = scanningIds.value.filter(id => id !== item.id);
    }
};

const scrapeItem = async (item: Child) => {
    scrapingIds.value.push(item.id);
    try {
        const response = await api.post("/library/song/scrape", { ids: [item.id], mode: 'full' });
        await handleUpdatedIds(response.data as string[]);
        toast.add({ severity: 'success', summary: 'Success', detail: 'Metadata scraped successfully', life: 3000 });
    } catch (error: any) {
        console.error("Failed to scrape item:", error);
        toast.add({ 
            severity: 'error', 
            summary: 'Scrape Failed', 
            detail: error.response?.data?.error || error.message || 'Failed to scrape metadata', 
            life: 5000 
        });
    } finally {
        scrapingIds.value = scrapingIds.value.filter(id => id !== item.id);
    }
};

const batchScan = () => {
    const itemsToScan = selectedItems.value;
    if (itemsToScan.length === 0) return;

    confirm.require({
        message: `Are you sure you want to scan ${itemsToScan.length} items?`,
        header: 'Confirm Batch Scan',
        icon: 'pi pi-refresh',
        rejectProps: {
            label: 'Cancel',
            severity: 'secondary',
            outlined: true
        },
        acceptProps: {
            label: 'Scan All',
            severity: 'primary'
        },
        accept: async () => {
            const ids = itemsToScan.map(item => item.id);
            // Add all to scanningIds for visual feedback
            ids.forEach(id => scanningIds.value.push(id));
            
            try {
                const response = await api.post("/library/scan", { ids });
                await handleUpdatedIds(response.data as string[]);
                
                selectedItems.value = [];
                toast.add({ severity: 'success', summary: 'Success', detail: `Scanned ${ids.length} items`, life: 3000 });
            } catch (error: any) {
                console.error("Batch scan failed:", error);
                toast.add({ 
                    severity: 'error', 
                    summary: 'Scan Failed', 
                    detail: error.response?.data?.error || error.message || 'Failed to scan items', 
                    life: 5000 
                });
            } finally {
                // Remove all from scanningIds
                scanningIds.value = scanningIds.value.filter(id => !ids.includes(id));
            }
        }
    });
};

const batchScrape = () => {
    const itemsToScrape = selectedItems.value.filter(item => !item.isDir);
    if (itemsToScrape.length === 0) return;

    confirm.require({
        message: `Are you sure you want to scrape metadata for ${itemsToScrape.length} songs? This will overwrite existing tags.`,
        header: 'Confirm Batch Scrape',
        icon: 'pi pi-search',
        rejectProps: {
            label: 'Cancel',
            severity: 'secondary',
            outlined: true
        },
        acceptProps: {
            label: 'Scrape All',
            severity: 'primary'
        },
        accept: async () => {
            const ids = itemsToScrape.map(item => item.id);
            // Add all to scrapingIds for visual feedback
            ids.forEach(id => scrapingIds.value.push(id));
            
            try {
                const response = await api.post("/library/song/scrape", { ids, mode: 'full' });
                await handleUpdatedIds(response.data as string[]);
                
                // Update selected items if they were affected (though they are usually cleared after batch)
                selectedItems.value = [];
                toast.add({ severity: 'success', summary: 'Success', detail: `Scraped ${ids.length} items`, life: 3000 });
            } catch (error: any) {
                console.error("Batch scrape failed:", error);
                toast.add({ 
                    severity: 'error', 
                    summary: 'Scrape Failed', 
                    detail: error.response?.data?.error || error.message || 'Failed to scrape items', 
                    life: 5000 
                });
            } finally {
                // Remove all from scrapingIds
                scrapingIds.value = scrapingIds.value.filter(id => !ids.includes(id));
            }
        }
    });
};
</script>

<template>
    <div class="p-4 h-full flex flex-col overflow-hidden">
        <Breadcrumb :model="breadcrumbs" class="mb-4 shrink-0" />

        <div v-if="!route.query.id" class="flex-1 overflow-y-auto">
            <h2 class="text-2xl font-bold mb-4">Music Folders</h2>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                <Card
                    v-for="folder in folders"
                    :key="folder.id"
                    class="cursor-pointer hover:shadow-lg transition-shadow"
                    @click="selectFolder(folder)"
                >
                    <template #title>
                        <div class="flex justify-between items-center">
                            <span>{{ folder.name }}</span>
                            <div class="flex gap-1">
                                <Button
                                    :icon="scrapingIds.includes(folder.directoryId) ? 'pi pi-spin pi-spinner' : 'pi pi-search-plus'"
                                    :disabled="scrapingIds.includes(folder.directoryId)"
                                    severity="secondary"
                                    variant="text"
                                    rounded
                                    size="small"
                                    @click.stop="scrapeFolder(folder)"
                                    v-tooltip="'Scrape Folder'"
                                />
                                <Button
                                    :icon="scanningIds.includes(folder.directoryId) ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'"
                                    :disabled="scanningIds.includes(folder.directoryId)"
                                    severity="secondary"
                                    variant="text"
                                    rounded
                                    size="small"
                                    @click.stop="scanFolder(folder)"
                                    v-tooltip="'Scan Folder'"
                                />
                            </div>
                        </div>
                    </template>
                    <template #content>
                        <p class="text-sm text-gray-500">{{ folder.path }}</p>
                    </template>
                </Card>
            </div>
        </div>

        <div v-else class="flex flex-1 gap-4 overflow-hidden">
            <!-- Left: File List -->
            <div class="flex-1 flex flex-col min-w-0 gap-4 overflow-hidden">
                <LibraryTable
                    :items="currentDir?.child || []"
                    :loading="loading"
                    :totalRecords="currentDir?.totalCount || 0"
                    v-model:first="first"
                    v-model:rows="rows"
                    :scanningIds="scanningIds"
                    :scrapingIds="scrapingIds"
                    v-model:isSelectionMode="isSelectionMode"
                    v-model:selection="selectionValue"
                    @row-click="onRowClick"
                    @row-dblclick="(e) => navigate(e.data)"
                    @scan="scanItem"
                    @scrape="scrapeItem"
                    @batch-scan="batchScan"
                    @batch-scrape="batchScrape"
                    @page="onPage"
                />
            </div>

            <!-- Right: Detailed View -->
            <div class="w-80 shrink-0 flex flex-col gap-4 overflow-y-auto">
                <LibraryDetail
                    :item="selectedSong"
                    :refreshKey="refreshKey"
                    @navigate="navigate"
                    @edit="editItem"
                />
            </div>
        </div>

        <MetadataDialog
            v-model:visible="editDialogVisible"
            :item="editingItem"
            @save="onMetadataSave"
        />
    </div>
</template>
