<script setup lang="ts">
const year = new Date().getFullYear();
</script>

<template>
    <main class="page">
        <h1>miko-web</h1>
        <p>Vite + Vue 3 + TypeScript</p>
        <small>© {{ year }}</small>
    </main>
</template>

<style scoped>
.page {
    min-height: 100vh;
    display: grid;
    place-content: center;
    gap: 8px;
    text-align: center;
}
</style>
