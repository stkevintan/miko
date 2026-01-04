<script setup lang="ts">
import Panel from "primevue/panel";
import Button from "primevue/button";
import CoverArt from "./CoverArt.vue";
import { Child } from "@/types/library";

defineProps<{
    item: Child | null;
    refreshKey: number;
}>();

const emit = defineEmits<{
    (e: "navigate", item: Child): void;
    (e: "edit", item: Child): void;
}>();

const formatSize = (bytes: number) => {
    if (!bytes) return "-";
    const k = 1024;
    const sizes = ["Bytes", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
};

// const formatDuration = (seconds: number) => {
//   if (!seconds) return '-';
//   const mins = Math.floor(seconds / 60);
//   const secs = seconds % 60;
//   return `${mins}:${secs.toString().padStart(2, '0')}`;
// };
</script>

<template>
    <Panel
        v-if="item"
        :header="item.isDir ? 'Directory Details' : 'Song Details'"
        class="h-full"
    >
        <div v-if="!item.isDir" class="flex flex-col gap-4">
            <CoverArt :coverArt="item.coverArt" :refreshKey="refreshKey" />
            <div class="grid grid-cols-1 gap-3 text-sm">
                <div>
                    <div
                        class="font-bold text-surface-500 uppercase text-xs mb-1"
                    >
                        Title
                    </div>
                    <div class="text-base font-semibold">{{ item.title }}</div>
                </div>
                <div>
                    <div
                        class="font-bold text-surface-500 uppercase text-xs mb-1"
                    >
                        Artist
                    </div>
                    <div>{{ item.artist || "-" }}</div>
                </div>
                <div>
                    <div
                        class="font-bold text-surface-500 uppercase text-xs mb-1"
                    >
                        Album
                    </div>
                    <div>{{ item.album || "-" }}</div>
                </div>
                <div class="grid grid-cols-2 gap-2">
                    <div>
                        <div
                            class="font-bold text-surface-500 uppercase text-xs mb-1"
                        >
                            Track
                        </div>
                        <div>{{ item.track || "-" }}</div>
                    </div>
                    <div>
                        <div
                            class="font-bold text-surface-500 uppercase text-xs mb-1"
                        >
                            Year
                        </div>
                        <div>{{ item.year || "-" }}</div>
                    </div>
                </div>
                <div>
                    <div
                        class="font-bold text-surface-500 uppercase text-xs mb-1"
                    >
                        Genre
                    </div>
                    <div>{{ item.genre || "-" }}</div>
                </div>
                <div class="grid grid-cols-2 gap-2">
                    <div>
                        <div
                            class="font-bold text-surface-500 uppercase text-xs mb-1"
                        >
                            Bitrate
                        </div>
                        <div>{{ item.bitRate }} kbps</div>
                    </div>
                    <div>
                        <div
                            class="font-bold text-surface-500 uppercase text-xs mb-1"
                        >
                            Format
                        </div>
                        <div>{{ item.suffix }}</div>
                    </div>
                </div>
                <div>
                    <div
                        class="font-bold text-surface-500 uppercase text-xs mb-1"
                    >
                        Size
                    </div>
                    <div>{{ formatSize(item.size || 0) }}</div>
                </div>
                <div>
                    <div
                        class="font-bold text-surface-500 uppercase text-xs mb-1"
                    >
                        Path
                    </div>
                    <div class="break-all text-xs opacity-70">
                        {{ item.path }}
                    </div>
                </div>
                <Button
                    label="Edit Metadata"
                    icon="pi pi-pencil"
                    class="w-full mt-2"
                    severity="contrast"
                    @click="emit('edit', item)"
                />
            </div>
        </div>
        <div v-else class="flex flex-col gap-4">
            <div
                class="w-full aspect-square bg-surface-100 dark:bg-surface-800 rounded-lg overflow-hidden flex items-center justify-center border border-surface-200 dark:border-surface-700"
            >
                <i class="pi pi-folder text-6xl text-yellow-500"></i>
            </div>
            <div>
                <div class="font-bold text-surface-500 uppercase text-xs mb-1">
                    Name
                </div>
                <div class="text-base font-semibold">{{ item.title }}</div>
            </div>
            <Button
                label="Open Directory"
                icon="pi pi-folder-open"
                class="w-full"
                @click="emit('navigate', item)"
            />
        </div>
    </Panel>
    <div
        v-else
        class="h-full flex flex-col items-center justify-center border-2 border-dashed border-surface-200 dark:border-surface-800 rounded-lg text-surface-400 p-6 text-center"
    >
        <i class="pi pi-info-circle text-4xl mb-4"></i>
        <p>Select a file or folder to view detailed information</p>
    </div>
</template>
