<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from '../stores/auth';
import { useRouter } from 'vue-router';
import Card from 'primevue/card';
import InputText from 'primevue/inputtext';
import Password from 'primevue/password';
import Button from 'primevue/button';
import Message from 'primevue/message';

const username = ref('');
const password = ref('');
const error = ref('');
const loading = ref(false);

const authStore = useAuthStore();
const router = useRouter();

const handleLogin = async () => {
  error.value = '';
  loading.value = true;
  try {
    await authStore.login(username.value, password.value);
    router.push('/');
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Login failed. Please check your credentials.';
  } finally {
    loading.value = false;
  }
};

import { useTheme } from '../composables/useTheme';

const { isDark, toggleDarkMode } = useTheme();
</script>

<template>
  <div class="flex justify-center items-center min-h-screen bg-surface-50 dark:bg-surface-950 p-4 relative">
    <div class="absolute top-4 right-4">
      <Button 
        :icon="isDark ? 'pi pi-sun' : 'pi pi-moon'" 
        variant="text" 
        severity="secondary" 
        rounded 
        @click="toggleDarkMode"
      />
    </div>
    <Card class="w-full max-w-md shadow-lg">
      <template #title>
        <h1 class="text-2xl font-bold text-center m-0 text-surface-900 dark:text-surface-0">Login to Miko</h1>
      </template>
      <template #content>
        <form @submit.prevent="handleLogin" class="flex flex-col gap-4 mt-4">
          <Message v-if="error" severity="error" variant="simple">{{ error }}</Message>
          
          <div class="flex flex-col gap-2">
            <label for="username" class="font-medium text-surface-700 dark:text-surface-300">Username</label>
            <InputText
              id="username"
              v-model="username"
              type="text"
              required
              :disabled="loading"
              class="w-full"
              placeholder="Enter your username"
            />
          </div>

          <div class="flex flex-col gap-2">
            <label for="password" class="font-medium text-surface-700 dark:text-surface-300">Password</label>
            <Password
              id="password"
              v-model="password"
              required
              :disabled="loading"
              :feedback="false"
              toggleMask
              fluid
              placeholder="Enter your password"
            />
          </div>

          <Button
            type="submit"
            label="Login"
            :loading="loading"
            class="w-full mt-2"
          />
        </form>
      </template>
    </Card>
  </div>
</template>

<style scoped>
/* Tailwind handles most styling now */
</style>
