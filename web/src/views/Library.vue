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

const fetchDirectory = async (id: string) => {
    loading.value = true;
    try {
        const response = await api.get(`/library/directory?id=${id}`);
        currentDir.value = response.data;
        updateBreadcrumbs();
    } catch (error) {
        console.error("Failed to fetch directory:", error);
    } finally {
        loading.value = false;
    }
};

const updateBreadcrumbs = () => {
    const items: BreadCrumbItem[] = [
        { label: "Library", command: () => void router.push("/library") },
    ];

    if (currentDir.value && route.query.id) {
        items.push({ label: currentDir.value.name });
    }

    breadcrumbs.value = items;
};

const navigate = (item: Child) => {
    if (item.isDir) {
        selectedSong.value = null;
        selectedItems.value = [];
        router.push({
            path: "/library",
            query: {
                id: item.id,
            },
        });
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
        await api.post("/library/scan", { id: folder.directoryId });
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

onMounted(async () => {
    await fetchFolders();
    handleRoute();
});

const handleRoute = () => {
    const { id } = route.query;
    selectedItems.value = [];
    if (typeof id === "string") {
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

const scanItem = async (item: Child) => {
    scanningIds.value.push(item.id);
    try {
        const response = await api.post("/library/scan", { id: item.id });
        // Update local state
        if (currentDir.value) {
            const index = currentDir.value.child.findIndex((c) => c.id === item.id);
            if (index !== -1) {
                currentDir.value.child[index] = response.data;
            }
        }
        if (selectedSong.value?.id === item.id) {
            selectedSong.value = response.data;
        }
        refreshKey.value = Date.now();
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
        const response = await api.post("/library/song/scrape", { id: item.id });
        // Update local state
        if (currentDir.value) {
            const index = currentDir.value.child.findIndex((c) => c.id === item.id);
            if (index !== -1) {
                currentDir.value.child[index] = response.data;
            }
        }
        if (selectedSong.value?.id === item.id) {
            selectedSong.value = response.data;
        }
        refreshKey.value = Date.now();
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

const deleteItem = (item: Child) => {
    confirm.require({
        message: `Are you sure you want to delete "${item.title}"? This will delete the file from disk.`,
        header: 'Confirm Deletion',
        icon: 'pi pi-exclamation-triangle',
        rejectProps: {
            label: 'Cancel',
            severity: 'secondary',
            outlined: true
        },
        acceptProps: {
            label: 'Delete',
            severity: 'danger'
        },
        accept: async () => {
            try {
                await api.post("/library/delete", { id: item.id });
                if (currentDir.value) {
                    currentDir.value.child = currentDir.value.child.filter(c => c.id !== item.id);
                }
                if (selectedSong.value?.id === item.id) {
                    selectedSong.value = null;
                }
                toast.add({ severity: 'success', summary: 'Success', detail: 'Item deleted successfully', life: 3000 });
            } catch (error: any) {
                console.error("Failed to delete item:", error);
                toast.add({ 
                    severity: 'error', 
                    summary: 'Delete Failed', 
                    detail: error.response?.data?.error || error.message || 'Failed to delete item', 
                    life: 5000 
                });
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
                await api.post("/library/song/scrape", { ids });
                
                // Refresh the directory to get updated data
                if (route.query.id) {
                    await fetchDirectory(route.query.id as string);
                }
                
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

const batchDelete = () => {
    const itemsToDelete = selectedItems.value;
    if (itemsToDelete.length === 0) return;

    confirm.require({
        message: `Are you sure you want to delete ${itemsToDelete.length} items? This will delete the files from disk.`,
        header: 'Confirm Batch Deletion',
        icon: 'pi pi-exclamation-triangle',
        rejectProps: {
            label: 'Cancel',
            severity: 'secondary',
            outlined: true
        },
        acceptProps: {
            label: 'Delete All',
            severity: 'danger'
        },
        accept: async () => {
            const ids = itemsToDelete.map(item => item.id);
            try {
                await api.post("/library/delete", { ids });
                
                if (currentDir.value) {
                    currentDir.value.child = currentDir.value.child.filter(c => !ids.includes(c.id));
                }
                if (selectedSong.value && ids.includes(selectedSong.value.id)) {
                    selectedSong.value = null;
                }
                
                selectedItems.value = [];
                toast.add({ severity: 'success', summary: 'Success', detail: `Deleted ${ids.length} items`, life: 3000 });
            } catch (error: any) {
                console.error("Batch delete failed:", error);
                toast.add({ 
                    severity: 'error', 
                    summary: 'Delete Failed', 
                    detail: error.response?.data?.error || error.message || 'Failed to delete items', 
                    life: 5000 
                });
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
                    :scanningIds="scanningIds"
                    :scrapingIds="scrapingIds"
                    v-model:isSelectionMode="isSelectionMode"
                    v-model:selection="selectionValue"
                    @row-click="onRowClick"
                    @row-dblclick="(e) => navigate(e.data)"
                    @edit="editItem"
                    @scan="scanItem"
                    @scrape="scrapeItem"
                    @delete="deleteItem"
                    @batch-scrape="batchScrape"
                    @batch-delete="batchDelete"
                />
            </div>

            <!-- Right: Detailed View -->
            <div class="w-80 shrink-0 flex flex-col gap-4 overflow-y-auto">
                <LibraryDetail
                    :item="selectedSong"
                    :refreshKey="refreshKey"
                    @navigate="navigate"
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
