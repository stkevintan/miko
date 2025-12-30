<script setup lang="ts">
import { useAuthStore } from '../stores/auth';
import Card from 'primevue/card';

const authStore = useAuthStore();
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold text-surface-900 dark:text-surface-0">Dashboard</h1>
    </div>

    <Card class="shadow-sm">
      <template #title>Welcome back, {{ authStore.user?.username }}!</template>
      <template #content>
        <p class="text-surface-600 dark:text-surface-400 m-0">
          This is your Miko dashboard. You can manage your music library and settings from the sidebar.
        </p>
      </template>
    </Card>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <Card v-for="i in 3" :key="i" class="shadow-sm">
        <template #title>Stat {{ i }}</template>
        <template #content>
          <div class="text-2xl font-bold">0</div>
          <p class="text-sm text-surface-500">Description for stat {{ i }}</p>
        </template>
      </Card>
    </div>
  </div>
</template>

<style scoped>
</style>
