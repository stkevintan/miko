<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useAuthStore } from '../stores/auth';
import api from '../api';
import Card from 'primevue/card';
import Button from 'primevue/button';
import InputText from 'primevue/inputtext';
import Password from 'primevue/password';
import Message from 'primevue/message';
import Tag from 'primevue/tag';
import Divider from 'primevue/divider';

const authStore = useAuthStore();
const oldPassword = ref('');
const newPassword = ref('');
const confirmPassword = ref('');
const loading = ref(false);
const error = ref('');
const success = ref('');

const handleChangePassword = async () => {
  error.value = '';
  success.value = '';
  
  if (newPassword.value !== confirmPassword.value) {
    error.value = 'New passwords do not match';
    return;
  }

  loading.value = true;
  try {
    await api.post('/change-password', {
      oldPassword: oldPassword.value,
      newPassword: newPassword.value
    });
    success.value = 'Password changed successfully';
    oldPassword.value = '';
    newPassword.value = '';
    confirmPassword.value = '';
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Failed to change password';
  } finally {
    loading.value = false;
  }
};

onMounted(async () => {
  if (!authStore.user) {
    await authStore.fetchUser();
  }
});

const roles = computed(() => [
  { label: 'Admin', value: authStore.user?.adminRole },
  { label: 'Settings', value: authStore.user?.settingsRole },
  { label: 'Download', value: authStore.user?.downloadRole },
  { label: 'Upload', value: authStore.user?.uploadRole },
  { label: 'Playlist', value: authStore.user?.playlistRole },
  { label: 'Stream', value: authStore.user?.streamRole },
  { label: 'Jukebox', value: authStore.user?.jukeboxRole },
  { label: 'Share', value: authStore.user?.shareRole },
]);
</script>

<template>
  <div class="max-w-4xl mx-auto flex flex-col gap-6 h-full overflow-y-auto">
    <h1 class="text-3xl font-bold text-surface-900 dark:text-surface-0">Profile</h1>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <!-- User Info & Roles -->
      <div class="md:col-span-2 flex flex-col gap-6">
        <Card class="shadow-sm">
          <template #title>Account Information</template>
          <template #content>
            <div class="flex flex-col gap-4">
              <div class="flex flex-col gap-1">
                <span class="text-sm text-surface-500">Username</span>
                <span class="text-lg font-medium">{{ authStore.user?.username }}</span>
              </div>
              <div class="flex flex-col gap-1">
                <span class="text-sm text-surface-500">Email</span>
                <span class="text-lg font-medium">{{ authStore.user?.email || 'Not set' }}</span>
              </div>
              
              <Divider />
              
              <div class="flex flex-col gap-3">
                <span class="text-sm text-surface-500">Roles & Permissions</span>
                <div class="flex flex-wrap gap-2">
                  <Tag 
                    v-for="role in roles" 
                    :key="role.label"
                    :value="role.label"
                    :severity="role.value ? 'success' : 'secondary'"
                    :icon="role.value ? 'pi pi-check' : 'pi pi-times'"
                  />
                </div>
              </div>
            </div>
          </template>
        </Card>
      </div>

      <!-- Change Password -->
      <div class="flex flex-col gap-6">
        <Card class="shadow-sm">
          <template #title>Security</template>
          <template #content>
            <form @submit.prevent="handleChangePassword" class="flex flex-col gap-4">
              <Message v-if="error" severity="error" variant="simple">{{ error }}</Message>
              <Message v-if="success" severity="success" variant="simple">{{ success }}</Message>
              
              <div class="flex flex-col gap-2">
                <label for="oldPassword" class="text-sm font-medium">Current Password</label>
                <Password 
                  id="oldPassword" 
                  v-model="oldPassword" 
                  :feedback="false" 
                  toggleMask 
                  fluid 
                  required
                />
              </div>

              <div class="flex flex-col gap-2">
                <label for="newPassword" class="text-sm font-medium">New Password</label>
                <Password 
                  id="newPassword" 
                  v-model="newPassword" 
                  toggleMask 
                  fluid 
                  required
                />
              </div>

              <div class="flex flex-col gap-2">
                <label for="confirmPassword" class="text-sm font-medium">Confirm New Password</label>
                <Password 
                  id="confirmPassword" 
                  v-model="confirmPassword" 
                  :feedback="false" 
                  toggleMask 
                  fluid 
                  required
                />
              </div>

              <Button 
                type="submit" 
                label="Update Password" 
                :loading="loading" 
                class="mt-2"
              />
            </form>
          </template>
        </Card>
      </div>
    </div>
  </div>
</template>
