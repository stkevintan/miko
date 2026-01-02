<script setup lang="ts">
import { ref, watch } from 'vue';
import Dialog from 'primevue/dialog';
import Button from 'primevue/button';
import InputText from 'primevue/inputtext';
import Divider from 'primevue/divider';
import CoverArt from './CoverArt.vue';
import api from '../api';
import { useAuthStore } from '../stores/auth';

const props = defineProps<{
  visible: boolean;
  item: any;
}>();

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void;
  (e: 'save', updatedItem: any): void;
}>();

const loading = ref(false);
const saving = ref(false);
const editingTags = ref<Array<{key: string, value: string}>>([]);
const coverFile = ref<File | null>(null);
const coverPreview = ref<string | null>(null);
const refreshKey = ref(Date.now());

const fetchTags = async () => {
  if (!props.item) return;
  loading.value = true;
  try {
    const response = await api.get(`/library/song/tags?id=${props.item.id}`);
    editingTags.value = Object.entries(response.data).flatMap(([key, values]) => 
      (values as string[]).map(value => ({ key, value }))
    );
    coverFile.value = null;
    coverPreview.value = null;
  } catch (error) {
    console.error('Failed to fetch tags:', error);
  } finally {
    loading.value = false;
  }
};

watch(() => props.visible, (newVal) => {
  if (newVal) {
    fetchTags();
  }
});

const onCoverChange = (file: File) => {
  coverFile.value = file;
  coverPreview.value = URL.createObjectURL(file);
};

const addTag = () => {
  editingTags.value.push({ key: '', value: '' });
};

const removeTag = (index: number) => {
  editingTags.value.splice(index, 1);
};

const saveEdit = async () => {
  if (!props.item) return;
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
      id: props.item.id,
      tags: tagsMap
    });

    let updatedItem = response.data;

    // Upload cover if changed
    if (coverFile.value) {
      const formData = new FormData();
      formData.append('id', props.item.id);
      formData.append('file', coverFile.value);
      const coverResponse = await api.post('/library/song/cover', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });
      updatedItem = coverResponse.data;
      refreshKey.value = Date.now();
    }
    
    emit('save', updatedItem);
    emit('update:visible', false);
  } catch (error) {
    console.error('Failed to save edit:', error);
  } finally {
    saving.value = false;
  }
};
</script>

<template>
  <Dialog 
    :visible="visible" 
    @update:visible="emit('update:visible', $event)"
    header="Edit Song Metadata" 
    :modal="true" 
    :style="{ width: '600px' }"
  >
    <div v-if="item" class="flex flex-col gap-4 pt-2">
      <div class="flex gap-4 items-start">
        <CoverArt 
          :coverArt="item.coverArt" 
          :previewUrl="coverPreview" 
          :refreshKey="refreshKey" 
          editable 
          sizeClass="w-32 h-32 shrink-0" 
          iconSize="text-4xl" 
          @change="onCoverChange" 
        />
        <div class="flex-1 flex flex-col gap-1 min-w-0">
          <div class="font-bold text-surface-500 uppercase text-xs mb-1">File Path</div>
          <div class="text-sm text-surface-600 break-all line-clamp-2 mb-2">{{ item.path }}</div>
          <div class="text-xs text-surface-400 italic">Click the image to upload a new cover art</div>
        </div>
      </div>

      <Divider />
      
      <div v-if="loading" class="flex justify-center p-4">
        <i class="pi pi-spin pi-spinner text-2xl"></i>
      </div>
      <div v-else class="max-h-[40vh] overflow-y-auto pr-2 flex flex-col gap-3">
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
        <Button icon="pi pi-plus" label="Add Tag" text severity="contrast" @click="addTag" class="whitespace-nowrap" :disabled="loading" />
        <div class="flex gap-2">
          <Button label="Cancel" icon="pi pi-times" text severity="secondary" @click="emit('update:visible', false)" :disabled="saving" />
          <Button label="Save" icon="pi pi-check" @click="saveEdit" :loading="saving" :disabled="loading" />
        </div>
      </div>
    </template>
  </Dialog>
</template>
