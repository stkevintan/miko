import { defineStore } from 'pinia';
import api from '../api';
export interface User {
  username: string
  email?: string
  adminRole: boolean
  settingsRole: boolean
  downloadRole: boolean
  uploadRole: boolean
  playlistRole: boolean
  streamRole: boolean
  jukeboxRole: boolean
  shareRole: boolean
}
export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token') || null,
    user: null as User | null,
  }),
  getters: {
    isAuthenticated: (state) => !!state.token,
  },
  actions: {
    async login(username: string, password: string) {
      try {
        const response = await api.post('/login', { username, password });
        const { token } = response.data;
        this.token = token;
        localStorage.setItem('token', token);
        await this.fetchUser();
        return true;
      } catch (error) {
        console.error('Login failed:', error);
        throw error;
      }
    },
    async fetchUser() {
      try {
        const response = await api.get('/me');
        this.user = response.data;
      } catch (error) {
        console.error('Failed to fetch user:', error);
        this.logout();
      }
    },
    logout() {
      this.token = null;
      this.user = null;
      localStorage.removeItem('token');
    },
  },
});
