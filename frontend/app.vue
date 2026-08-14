<script setup lang="ts">
import { computed } from 'vue'

const route = useRoute()
const isAdministrationRoute = computed(() => route.path.startsWith('/admin'))
</script>

<template>
  <div class="min-h-screen bg-zinc-950 text-zinc-100">
    <a
      v-if="!isAdministrationRoute"
      href="#main-content"
      class="fixed left-4 top-4 z-[100] -translate-y-24 bg-violet-200 px-4 py-2 text-sm font-semibold text-zinc-950 transition-transform focus:translate-y-0"
    >跳至主要内容</a>
    <ArchiveGridCanvas v-if="!isAdministrationRoute" />
    <TargetCursor v-if="!isAdministrationRoute" />
    <SiteNavigation v-if="!isAdministrationRoute" />
    <GlobalAudioPlayer v-if="!isAdministrationRoute" />
    <PageTransitionOverlay />
    <div id="main-content" tabindex="-1" class="relative z-10" :class="{ 'pt-24': !isAdministrationRoute }">
      <NuxtPage />
    </div>
  </div>
</template>
