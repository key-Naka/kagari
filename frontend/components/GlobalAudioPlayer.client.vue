<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, shallowRef, useTemplateRef, watch } from 'vue'

const route = useRoute()
const player = usePlayerStore()
const audioElement = useTemplateRef<HTMLAudioElement>('audio')

let audioContext: AudioContext | null = null
let analyser: AnalyserNode | null = null
let animationFrame = 0
let playbackOperation = 0
let reducedMotionQuery: MediaQueryList | null = null
let reducedMotionListener: (() => void) | null = null
const prefersReducedMotion = shallowRef(false)

const showMiniPlayer = computed(() => (
  player.hasStarted
  && player.currentTrack !== null
  && route.path !== '/music'
  && !route.path.startsWith('/admin')
))

const progressMaximum = computed(() => Math.max(1, player.durationMs))
const progressValue = computed(() => Math.min(player.currentTimeMs, progressMaximum.value))
const shouldSampleSpectrum = computed(() => route.path === '/music' && !prefersReducedMotion.value)

watch(
  () => [player.currentTrack?.id ?? null, player.isPlaying] as const,
  async ([trackId, shouldPlay], [previousTrackId]) => {
    const operation = ++playbackOperation
    const element = audioElement.value
    const track = player.currentTrack
    if (!element || !track || trackId === null) return

    if (trackId !== previousTrackId || element.src !== track.audio.publicUrl) {
      element.src = track.audio.publicUrl
      element.load()
    }

    if (!shouldPlay) {
      element.pause()
      stopSpectrum()
      return
    }

    try {
      await ensureAudioGraph(element)
      if (operation !== playbackOperation || !player.isPlaying || player.currentTrack?.id !== trackId) return
      await element.play()
      if (operation !== playbackOperation || !player.isPlaying || player.currentTrack?.id !== trackId) element.pause()
    } catch {
      if (operation === playbackOperation) player.reportPlaybackError('音频暂时无法播放，请稍后重试。')
    }
  },
  { flush: 'post' },
)

function handlePlay(): void {
  player.syncPlaying(true)
  if (shouldSampleSpectrum.value) startSpectrum()
}

function handlePlaybackError(): void {
  stopSpectrum()
  player.reportPlaybackError('音频加载失败，请检查网络或稍后重试。')
}

function handlePause(): void {
  player.syncPlaying(false)
  stopSpectrum()
}

function handleTimeUpdate(event: Event): void {
  player.syncProgress((event.currentTarget as HTMLAudioElement).currentTime)
}

function seek(event: Event): void {
  const element = audioElement.value
  if (!element) return
  const milliseconds = Number((event.currentTarget as HTMLInputElement).value)
  element.currentTime = milliseconds / 1000
  player.syncProgress(element.currentTime)
}

async function ensureAudioGraph(element: HTMLAudioElement): Promise<void> {
  if (!audioContext) {
    audioContext = new AudioContext()
    analyser = audioContext.createAnalyser()
    analyser.fftSize = 64
    const source = audioContext.createMediaElementSource(element)
    source.connect(analyser)
    analyser.connect(audioContext.destination)
  }
  if (audioContext.state === 'suspended') {
    await audioContext.resume()
  }
}

function startSpectrum(): void {
  if (!analyser || animationFrame !== 0 || !shouldSampleSpectrum.value) return
  const values = new Uint8Array(analyser.frequencyBinCount)
  const sample = () => {
    if (!analyser || !player.isPlaying) {
      animationFrame = 0
      return
    }
    analyser.getByteFrequencyData(values)
    player.setSpectrum(values)
    animationFrame = requestAnimationFrame(sample)
  }
  animationFrame = requestAnimationFrame(sample)
}

watch(
  () => [player.isPlaying, route.path, prefersReducedMotion.value] as const,
  ([isPlaying]) => {
    if (isPlaying && shouldSampleSpectrum.value) startSpectrum()
    else stopSpectrum()
  },
)

onMounted(() => {
  reducedMotionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  reducedMotionListener = () => { prefersReducedMotion.value = reducedMotionQuery?.matches ?? false }
  reducedMotionListener()
  reducedMotionQuery.addEventListener('change', reducedMotionListener)
})

function stopSpectrum(): void {
  if (animationFrame !== 0) cancelAnimationFrame(animationFrame)
  animationFrame = 0
  player.setSpectrum(Array.from({ length: 32 }, () => 0))
}

function formatTime(milliseconds: number): string {
  const totalSeconds = Math.max(0, Math.floor(milliseconds / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = String(totalSeconds % 60).padStart(2, '0')
  return `${minutes}:${seconds}`
}

onBeforeUnmount(() => {
  playbackOperation++
  stopSpectrum()
  audioElement.value?.pause()
  void audioContext?.close()
  if (reducedMotionListener) reducedMotionQuery?.removeEventListener('change', reducedMotionListener)
})
</script>

<template>
  <audio
    ref="audio"
    crossorigin="anonymous"
    preload="metadata"
    @play="handlePlay"
    @pause="handlePause"
    @ended="player.playNext"
    @error="handlePlaybackError"
    @timeupdate="handleTimeUpdate"
  />

  <Transition name="mini-player">
    <aside
      v-if="showMiniPlayer && player.currentTrack"
      class="mini-player"
      data-testid="mini-player"
      aria-label="Mini Player"
    >
      <div class="mini-player__art" aria-hidden="true">
        <img :src="player.currentTrack.cover.publicUrl" alt="" width="88" height="140">
        <span class="mini-player__pulse" :class="{ 'mini-player__pulse--active': player.isPlaying }" />
      </div>

      <div class="mini-player__body">
        <div class="mini-player__meta">
          <div>
            <p class="mini-player__eyebrow">NOW RESONATING</p>
            <p class="mini-player__title">{{ player.currentTrack.title }}</p>
          </div>
          <span class="mini-player__time">{{ formatTime(player.currentTimeMs) }} / {{ formatTime(player.durationMs) }}</span>
        </div>

        <input
          class="mini-player__progress"
          type="range"
          name="playback-progress"
          min="0"
          :max="progressMaximum"
          :value="progressValue"
          aria-label="播放进度"
          @input="seek"
        >

        <div class="mini-player__controls">
          <button type="button" class="cursor-target" aria-label="上一首" @click="player.playPrevious">←</button>
          <button
            type="button"
            class="mini-player__primary cursor-target"
            :aria-label="player.isPlaying ? '暂停' : '继续播放'"
            @click="player.togglePlayback"
          >
            {{ player.isPlaying ? 'Ⅱ' : '▶' }}
          </button>
          <button type="button" class="cursor-target" aria-label="下一首" @click="player.playNext">→</button>
          <NuxtLink class="mini-player__return cursor-target" to="/music">返回音乐页</NuxtLink>
        </div>
        <p v-if="player.errorMessage" class="mini-player__error" role="alert">{{ player.errorMessage }}</p>
      </div>
    </aside>
  </Transition>
</template>

<style scoped>
.mini-player {
  position: fixed;
  right: max(clamp(1rem, 3vw, 2.5rem), env(safe-area-inset-right));
  bottom: max(clamp(1rem, 3vw, 2.5rem), env(safe-area-inset-bottom));
  z-index: 45;
  display: grid;
  grid-template-columns: 5.5rem minmax(0, 1fr);
  width: min(30rem, calc(100vw - 2rem));
  overflow: hidden;
  border: 1px solid rgb(113 113 122 / 0.55);
  background: linear-gradient(135deg, rgb(9 9 11 / 0.96), rgb(24 24 27 / 0.9));
  box-shadow: 0 1.5rem 4rem rgb(0 0 0 / 0.55), inset 0 1px rgb(255 255 255 / 0.06);
  backdrop-filter: blur(18px);
}

.mini-player__art { position: relative; min-height: 8.75rem; background: #18181b; }
.mini-player__art img { width: 100%; height: 100%; object-fit: cover; filter: saturate(0.75) contrast(1.08); }
.mini-player__pulse { position: absolute; inset: 50%; width: 0.55rem; height: 0.55rem; border-radius: 999px; background: #c4b5fd; box-shadow: 0 0 1.5rem #8b5cf6; transform: translate(-50%, -50%); }
.mini-player__pulse--active { animation: resonate 1.6s ease-out infinite; }
.mini-player__body { min-width: 0; padding: 1rem 1.1rem; }
.mini-player__meta { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; }
.mini-player__eyebrow { color: #8b5cf6; font-size: 0.58rem; letter-spacing: 0.22em; }
.mini-player__title { margin-top: 0.3rem; overflow: hidden; color: #fafafa; font-family: Georgia, serif; font-size: 1.05rem; text-overflow: ellipsis; white-space: nowrap; }
.mini-player__time { flex: none; color: #71717a; font-family: ui-monospace, monospace; font-size: 0.65rem; font-variant-numeric: tabular-nums; }
.mini-player__progress { width: 100%; height: 1.5rem; margin-top: 0.2rem; accent-color: #8b5cf6; cursor: pointer; }
.mini-player__controls { display: flex; align-items: center; gap: 0.45rem; margin-top: 0.7rem; }
.mini-player__controls button { display: grid; width: 2rem; height: 2rem; place-items: center; border: 1px solid #3f3f46; color: #d4d4d8; touch-action: manipulation; transition: border-color 160ms ease, color 160ms ease; }
.mini-player__controls button:hover { border-color: #a78bfa; color: #fff; }
.mini-player__controls button:focus-visible, .mini-player__return:focus-visible, .mini-player__progress:focus-visible { outline: 2px solid #c4b5fd; outline-offset: 3px; }
.mini-player__primary { background: #ede9fe; color: #18181b !important; }
.mini-player__return { margin-left: auto; color: #a1a1aa; font-size: 0.7rem; text-decoration: none; }
.mini-player__return:hover { color: #fff; }
.mini-player__error { margin-top: 0.65rem; color: #fecdd3; font-size: 0.7rem; line-height: 1.4; }
.mini-player-enter-active, .mini-player-leave-active { transition: opacity 220ms ease, transform 220ms ease; }
.mini-player-enter-from, .mini-player-leave-to { opacity: 0; transform: translateY(1rem); }

@keyframes resonate {
  0% { box-shadow: 0 0 0 0 rgb(139 92 246 / 0.65), 0 0 1.5rem #8b5cf6; }
  100% { box-shadow: 0 0 0 1.5rem rgb(139 92 246 / 0), 0 0 1.5rem #8b5cf6; }
}

@media (max-width: 540px) {
  .mini-player { grid-template-columns: 4.5rem minmax(0, 1fr); }
  .mini-player__art { min-height: 7.7rem; }
  .mini-player__time { display: none; }
}

@media (prefers-reduced-motion: reduce) {
  .mini-player__pulse--active { animation: none; }
  .mini-player-enter-active, .mini-player-leave-active { transition-duration: 1ms; }
}
</style>
