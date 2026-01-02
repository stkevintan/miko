<script setup lang="ts">
import { ref, onMounted, watch, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
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
    } catch (error) {
        console.error("Failed to scan folder:", error);
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
    } catch (error) {
        console.error("Failed to scan item:", error);
    } finally {
        scanningIds.value = scanningIds.value.filter(id => id !== item.id);
    }
};

const scrapeItem = (item: Child) => {
    console.log("Scrape item:", item);
    // TODO: Implement scrape logic
};

const deleteItem = (item: Child) => {
    console.log("Delete item:", item);
    // TODO: Implement delete logic
};

const batchScrape = () => {
    console.log("Batch scrape:", selectedItems.value);
};

const batchDelete = () => {
    console.log("Batch delete:", selectedItems.value);
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
