<script setup lang="ts">
import { computed } from 'vue'

const route = useRoute()
const isAdministrationRoute = computed(() => route.path.startsWith('/admin'))
</script>

<template>
  <div class="min-h-screen bg-zinc-950 text-zinc-100">
    <ArchiveGridCanvas v-if="!isAdministrationRoute" />
    <TargetCursor v-if="!isAdministrationRoute" />
    <SiteNavigation v-if="!isAdministrationRoute" />
    <PageTransitionOverlay />
    <div class="relative z-10" :class="{ 'pt-24': !isAdministrationRoute }">
      <NuxtPage />
    </div>
  </div>
</template>
