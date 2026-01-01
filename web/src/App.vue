<script setup lang="ts">
import { useTheme } from './composables/useTheme';

useTheme();
</script>

<template>
  <router-view />
</template>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
  height: 100vh;
  width: 100vw;
  overflow: auto;
}

body {
  margin: 0;
  padding: 0;
}
</style>
