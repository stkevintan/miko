<script setup lang="ts">
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import Button from 'primevue/button';

const props = defineProps<{
  items: any[];
  loading: boolean;
  selectionMode: 'single' | 'multiple';
  selection: any;
}>();

const emit = defineEmits<{
  (e: 'update:selection', value: any): void;
  (e: 'row-click', event: any): void;
  (e: 'row-dblclick', event: any): void;
  (e: 'edit', item: any): void;
  (e: 'scrape', item: any): void;
  (e: 'delete', item: any): void;
}>();

const formatDuration = (seconds: number) => {
  if (!seconds) return '-';
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
};
</script>

<template>
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
      :selectionMode="selectionMode"
      dataKey="id"
      @row-click="emit('row-click', $event)"
      @row-dblclick="emit('row-dblclick', $event)"
      paginator
      :rows="50"
      :rowsPerPageOptions="[20, 50, 100]"
      paginatorTemplate="FirstPageLink PrevPageLink PageLinks NextPageLink LastPageLink CurrentPageReport RowsPerPageDropdown"
      currentPageReportTemplate="{first} to {last} of {totalRecords}"
    >
      <Column v-if="selectionMode === 'multiple'" selectionMode="multiple" headerStyle="width: 3rem"></Column>
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
          <div class="flex gap-1 justify-end items-center">
            <Button v-if="!slotProps.data.isDir" icon="pi pi-pencil" variant="text" severity="secondary" rounded size="small" @click.stop="emit('edit', slotProps.data)" v-tooltip="'Edit'" />
            <Button icon="pi pi-search" variant="text" severity="secondary" rounded size="small" @click.stop="emit('scrape', slotProps.data)" v-tooltip="'Scrape'" />
            <Button icon="pi pi-trash" variant="text" severity="danger" rounded size="small" @click.stop="emit('delete', slotProps.data)" v-tooltip="'Delete'" />
          </div>
        </template>
      </Column>
    </DataTable>
  </div>
</template>
