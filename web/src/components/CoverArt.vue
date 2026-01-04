<script setup lang="ts">
import { ref, watch } from 'vue';
import api from '../api';
import { useAuthStore } from '../stores/auth';

const props = defineProps<{
  coverArt?: string;
  previewUrl?: string | null;
  refreshKey?: number;
  editable?: boolean;
  sizeClass?: string;
  iconSize?: string;
}>();

const emit = defineEmits<{
  (e: 'change', file: File): void;
}>();

const authStore = useAuthStore();
const coverArtError = ref(false);
const coverArtLoaded = ref(false);
const coverInput = ref<HTMLInputElement | null>(null);

watch(() => [props.coverArt, props.previewUrl], () => {
  coverArtError.value = false;
  coverArtLoaded.value = false;
});

const getCoverArtUrl = (id: string) => {
  if (!id) return '';
  return `${api.defaults.baseURL}/library/coverArt?id=${id}&token=${authStore.token}&t=${props.refreshKey || 0}`;
};

const triggerUpload = () => {
  if (props.editable) {
    coverInput.value?.click();
  }
};

const onFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  if (target.files && target.files[0]) {
    emit('change', target.files[0]);
  }
};
</script>

<template>
  <div 
    :class="[
      sizeClass || 'w-full aspect-square',
      'bg-surface-100 dark:bg-surface-800 rounded-lg overflow-hidden flex items-center justify-center border border-surface-200 dark:border-surface-700 relative',
      editable ? 'group cursor-pointer' : ''
    ]"
    @click="triggerUpload"
  >
    <i :class="['pi pi-headphones text-surface-400', iconSize || 'text-6xl']"></i>
    
    <img 
      v-if="(previewUrl || coverArt) && !coverArtError" 
      v-show="coverArtLoaded || previewUrl"
      :src="previewUrl || getCoverArtUrl(coverArt!)" 
      class="absolute inset-0 w-full h-full object-cover" 
      alt="Cover Art" 
      @load="coverArtLoaded = true"
      @error="coverArtError = true"
    />

    <template v-if="editable">
      <div class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 flex items-center justify-center transition-opacity">
        <i class="pi pi-camera text-white text-2xl"></i>
      </div>
      <input type="file" ref="coverInput" class="hidden" accept="image/*" @change="onFileChange" />
    </template>
  </div>
</template>
