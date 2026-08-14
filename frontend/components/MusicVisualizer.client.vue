<script setup lang="ts">
const player = usePlayerStore()
</script>

<template>
  <div
    class="music-visualizer"
    data-testid="music-visualizer"
    :data-active="String(player.isPlaying)"
    aria-hidden="true"
  >
    <div class="music-visualizer__halo music-visualizer__halo--outer" />
    <div class="music-visualizer__halo music-visualizer__halo--inner" />
    <div class="music-visualizer__bars">
      <span
        v-for="(value, index) in player.spectrum"
        :key="index"
        class="music-visualizer__bar"
        :style="{ height: `${Math.max(5, value / 2.55)}%` }"
      />
    </div>
  </div>
</template>

<style scoped>
.music-visualizer { position: relative; display: grid; width: min(34rem, 82vw); aspect-ratio: 1; place-items: center; }
.music-visualizer::before { position: absolute; width: 26%; aspect-ratio: 1; border: 1px solid rgb(196 181 253 / 0.45); border-radius: 999px; background: radial-gradient(circle, rgb(139 92 246 / 0.2), transparent 68%); box-shadow: 0 0 4rem rgb(124 58 237 / 0.18); content: ''; }
.music-visualizer__halo { position: absolute; border: 1px solid rgb(139 92 246 / 0.22); border-radius: 999px; }
.music-visualizer__halo--outer { inset: 8%; }
.music-visualizer__halo--inner { inset: 25%; border-style: dashed; }
.music-visualizer[data-active='true'] .music-visualizer__halo--outer { animation: breathe 2.4s ease-in-out infinite; }
.music-visualizer[data-active='true'] .music-visualizer__halo--inner { animation: breathe 1.7s ease-in-out infinite reverse; }
.music-visualizer__bars { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; gap: clamp(2px, 0.5vw, 6px); transform: rotate(-90deg); }
.music-visualizer__bar { width: clamp(2px, 0.45vw, 5px); max-height: 42%; border-radius: 99px; background: linear-gradient(to top, rgb(39 39 42 / 0), #7c3aed 42%, #ddd6fe); box-shadow: 0 0 0.8rem rgb(139 92 246 / 0.35); transition: height 80ms linear; transform-origin: center; }
@keyframes breathe { 50% { opacity: 0.35; transform: scale(0.96); } }
@media (prefers-reduced-motion: reduce) { .music-visualizer__halo { animation: none !important; } .music-visualizer__bar { transition: none; } }
</style>
