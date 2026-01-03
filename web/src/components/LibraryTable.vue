<script setup lang="ts">
import DataTable, { DataTableRowClickEvent } from 'primevue/datatable';
import Column from 'primevue/column';
import Button from 'primevue/button';
import Divider from 'primevue/divider';
import { Child } from '@/types/library';

const props = defineProps<{
  items: Child[];
  loading: boolean;
  isSelectionMode: boolean;
  selection: Child | Child[] | null;
  scanningIds?: string[];
  scrapingIds?: string[];
}>();

const emit = defineEmits<{
  (e: 'update:selection', value: Child | Child[] | null): void;
  (e: 'update:isSelectionMode', value: boolean): void;
  (e: 'row-click', event: DataTableRowClickEvent<Child>): void;
  (e: 'row-dblclick', event: DataTableRowClickEvent<Child>): void;
  (e: 'scan', item: Child): void;
  (e: 'scrape', item: Child): void;
  (e: 'batch-scan'): void;
  (e: 'batch-scrape'): void;
}>();

const toggleSelectionMode = () => {
  emit('update:isSelectionMode', !props.isSelectionMode);
};

const clearSelection = () => {
  emit('update:selection', props.isSelectionMode ? [] : null);
};

const formatDuration = (seconds: number) => {
  if (!seconds) return '-';
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
};
</script>

<template>
    <!-- Action Bar -->
    <div
      class="flex items-center justify-between p-2 bg-surface-100 dark:bg-surface-800 border border-surface-200 dark:border-surface-700 rounded-lg shrink-0"
    >
      <div class="flex items-center gap-2 px-2">
        <Button
          :icon="isSelectionMode ? 'pi pi-check-square' : 'pi pi-list'"
          :label="isSelectionMode ? 'Selection Mode' : 'Browse Mode'"
          size="small"
          variant="text"
          :severity="isSelectionMode ? 'primary' : 'secondary'"
          @click="toggleSelectionMode"
        />
        <template v-if="isSelectionMode && Array.isArray(selection) && selection.length > 0">
          <Divider layout="vertical" class="mx-0 h-4" />
          <span class="text-sm font-medium">{{ selection.length }} selected</span>
          <Button
            icon="pi pi-times"
            variant="text"
            severity="secondary"
            rounded
            size="small"
            @click="clearSelection"
          />
        </template>
      </div>
      <div v-if="isSelectionMode && Array.isArray(selection) && selection.length > 0" class="flex gap-2">
        <Button
          label="Scan"
          icon="pi pi-refresh"
          size="small"
          severity="secondary"
          @click="emit('batch-scan')"
        />
        <Button
          label="Scrape"
          icon="pi pi-search"
          size="small"
          severity="secondary"
          @click="emit('batch-scrape')"
        />
      </div>
    </div>

    <div class="flex-1 flex flex-col min-w-0 border border-surface-200 dark:border-surface-800 rounded-lg bg-surface-0 dark:bg-surface-900 overflow-hidden">
      <DataTable
        :value="items" 
        :loading="loading" 
        scrollable
        scrollHeight="flex"
        resizableColumns
        class="p-datatable-sm flex-1"
        :selection="selection"
        @update:selection="emit('update:selection', $event)"
        :selectionMode="isSelectionMode ? 'multiple' : 'single'"
        dataKey="id"
        @row-click="emit('row-click', $event)"
        @row-dblclick="emit('row-dblclick', $event)"
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
      <Column header="Actions" style="width: 4rem" frozen alignFrozen="right">
        <template #body="slotProps">
          <div class="flex gap-1 justify-end items-center">
            <Button 
              :icon="scanningIds?.includes(slotProps.data.id) ? 'pi pi-spin pi-spinner' : 'pi pi-refresh'" 
              :disabled="scanningIds?.includes(slotProps.data.id)"
              variant="text" 
              severity="secondary" 
              rounded 
              size="small" 
              @click.stop="emit('scan', slotProps.data)" 
              v-tooltip="'Scan'" 
            />
            <Button 
              :icon="scrapingIds?.includes(slotProps.data.id) ? 'pi pi-spin pi-spinner' : 'pi pi-search'" 
              :disabled="scrapingIds?.includes(slotProps.data.id)"
              variant="text" 
              severity="secondary" 
              rounded 
              size="small" 
              @click.stop="emit('scrape', slotProps.data)" 
              v-tooltip="'Scrape'" 
            />
          </div>
        </template>
      </Column>
    </DataTable>
  </div>
</template>
