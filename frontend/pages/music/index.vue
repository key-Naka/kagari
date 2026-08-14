<script setup lang="ts">
import { computed, watch } from 'vue'
import type { PublicTrack, TrackMedia } from '~/stores/player'

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const player = usePlayerStore()

const { data, status, error, refresh } = await useAsyncData('public-tracks', async () => {
  const response = await $fetch<unknown>(`${apiBase}/api/v1/tracks`)
  const parsed = parsePublicTracks(response)
  if (!parsed) throw new Error('音乐档案的数据格式无效。')
  return parsed
}, { default: () => [] })

watch(data, tracks => player.hydrateTracks(tracks), { immediate: true })

const isLoading = computed(() => status.value === 'pending')
const errorMessage = computed(() => error.value instanceof Error ? error.value.message : '音乐档案暂时无法读取。')

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function parseMedia(value: unknown, kind: 'image' | 'audio'): TrackMedia | null {
  if (!isRecord(value)) return null
  if (
    typeof value.id !== 'number'
    || typeof value.objectKey !== 'string'
    || typeof value.publicUrl !== 'string'
    || value.kind !== kind
    || typeof value.mimeType !== 'string'
    || typeof value.size !== 'number'
    || typeof value.originalName !== 'string'
  ) return null
  const width = typeof value.width === 'number' ? value.width : 0
  const height = typeof value.height === 'number' ? value.height : 0
  const durationMs = typeof value.durationMs === 'number' ? value.durationMs : 0
  if (kind === 'image' && (width <= 0 || height <= 0)) return null
  if (kind === 'audio' && durationMs <= 0) return null
  return {
    id: value.id,
    objectKey: value.objectKey,
    publicUrl: value.publicUrl,
    kind,
    mimeType: value.mimeType,
    size: value.size,
    originalName: value.originalName,
    width,
    height,
    durationMs,
  }
}

function parseTrack(value: unknown): PublicTrack | null {
  if (!isRecord(value) || typeof value.id !== 'number' || typeof value.title !== 'string' || typeof value.sortOrder !== 'number') return null
  const cover = parseMedia(value.cover, 'image')
  const audio = parseMedia(value.audio, 'audio')
  return cover && audio ? { id: value.id, title: value.title, cover, audio, sortOrder: value.sortOrder } : null
}

function parsePublicTracks(value: unknown): PublicTrack[] | null {
  if (!Array.isArray(value)) return null
  const tracks = value.map(parseTrack)
  return tracks.every((track): track is PublicTrack => track !== null) ? tracks : null
}

function toggleTrack(track: PublicTrack): void {
  if (player.currentTrackId === track.id && player.isPlaying) {
    player.pause()
    return
  }
  player.playTrack(track.id)
}

function formatDuration(milliseconds: number): string {
  const totalSeconds = Math.max(0, Math.round(milliseconds / 1000))
  return `${Math.floor(totalSeconds / 60)}:${String(totalSeconds % 60).padStart(2, '0')}`
}

useSeoMeta({
  title: '音乐档案 · Kagari',
  description: '聆听 Kagari 收录的独立音乐作品。',
})
</script>

<template>
  <main class="music-page">
    <div class="music-page__grain" aria-hidden="true" />
    <div class="music-page__shell">
      <NuxtLink to="/" class="music-page__back cursor-target">← 返回首页</NuxtLink>

      <header class="music-page__header">
        <div>
          <p class="music-page__index">ARCHIVE / 03 · RESONANCE</p>
          <h1>音乐档案</h1>
          <p class="music-page__intro">被保存下来的声响、余烬与夜间信号。选择一段录音后，波纹才会苏醒。</p>
        </div>
        <div class="music-page__summary">
          <span>{{ data.length.toString().padStart(2, '0') }}</span>
          <small>ENABLED TRACKS</small>
          <button type="button" :disabled="isLoading" @click="refresh()">{{ isLoading ? '读取中…' : '刷新档案' }}</button>
        </div>
      </header>

      <p v-if="isLoading && data.length === 0" class="music-page__notice" role="status">正在读取音乐档案…</p>
      <p v-else-if="error" class="music-page__notice music-page__notice--error" role="alert">{{ errorMessage }}</p>
      <p v-else-if="data.length === 0" class="music-page__notice">尚未启用任何 Track。</p>

      <section v-else class="music-stage" aria-label="音乐播放器">
        <div class="music-stage__visual">
          <MusicVisualizer />
          <div v-if="player.currentTrack" class="music-stage__caption">
            <span>{{ player.isPlaying ? 'SIGNAL ACTIVE' : 'SIGNAL DORMANT' }}</span>
            <strong>{{ player.currentTrack.title }}</strong>
          </div>
        </div>

        <ol class="track-list" aria-label="Track 列表">
          <li v-for="(track, index) in data" :key="track.id">
            <button
              type="button"
              class="track-card cursor-target"
              :class="{ 'track-card--current': player.currentTrackId === track.id }"
              :aria-label="`${player.currentTrackId === track.id && player.isPlaying ? '暂停' : '播放'} ${track.title}`"
              @click="toggleTrack(track)"
            >
              <span class="track-card__number">{{ String(index + 1).padStart(2, '0') }}</span>
              <span class="track-card__cover"><img :src="track.cover.publicUrl" :alt="`${track.title} 封面`" width="72" height="72" loading="lazy"></span>
              <span class="track-card__copy">
                <strong>{{ track.title }}</strong>
                <small>{{ track.audio.originalName }}</small>
              </span>
              <span class="track-card__duration">{{ formatDuration(track.audio.durationMs) }}</span>
              <span class="track-card__action">{{ player.currentTrackId === track.id && player.isPlaying ? 'Ⅱ' : '▶' }}</span>
            </button>
          </li>
        </ol>
      </section>

      <p v-if="player.errorMessage" class="music-page__notice music-page__notice--error" role="alert">{{ player.errorMessage }}</p>
    </div>
  </main>
</template>

<style scoped>
.music-page { position: relative; min-height: 100vh; overflow: hidden; background: radial-gradient(circle at 18% 12%, rgb(76 29 149 / 0.13), transparent 34rem), #09090b; color: #f4f4f5; }
.music-page__grain { position: absolute; inset: 0; opacity: 0.16; background-image: repeating-linear-gradient(0deg, transparent, transparent 3px, rgb(255 255 255 / 0.018) 4px); pointer-events: none; }
.music-page__shell { position: relative; width: min(78rem, calc(100% - 2rem)); margin: 0 auto; padding: 3rem 0 8rem; }
.music-page__back { color: #71717a; font-size: 0.78rem; letter-spacing: 0.08em; text-decoration: none; transition: color 160ms ease; }
.music-page__back:hover { color: #e4e4e7; }
.music-page__back:focus-visible, .music-page__summary button:focus-visible, .track-card:focus-visible { outline: 2px solid #c4b5fd; outline-offset: 4px; }
.music-page__header { display: flex; align-items: end; justify-content: space-between; gap: 3rem; margin-top: 2.5rem; padding-bottom: 2.25rem; border-bottom: 1px solid #27272a; }
.music-page__index { color: #8b5cf6; font-size: 0.66rem; letter-spacing: 0.28em; }
.music-page__header h1 { margin-top: 0.8rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3rem, 8vw, 7.2rem); font-weight: 400; letter-spacing: -0.055em; line-height: 0.88; text-wrap: balance; }
.music-page__intro { max-width: 34rem; margin-top: 1.5rem; color: #a1a1aa; font-size: 0.95rem; line-height: 1.8; }
.music-page__summary { display: grid; min-width: 9rem; justify-items: end; }
.music-page__summary span { font-family: Georgia, serif; font-size: 3.5rem; line-height: 1; }
.music-page__summary small { margin-top: 0.5rem; color: #52525b; font-size: 0.58rem; letter-spacing: 0.24em; }
.music-page__summary button { margin-top: 1.4rem; border-bottom: 1px solid #52525b; padding-bottom: 0.25rem; color: #a1a1aa; font-size: 0.72rem; }
.music-page__notice { margin-top: 2rem; border: 1px solid #27272a; background: rgb(24 24 27 / 0.55); padding: 1rem; color: #a1a1aa; }
.music-page__notice--error { border-color: rgb(244 63 94 / 0.38); color: #fecdd3; }
.music-stage { display: grid; grid-template-columns: minmax(0, 1.1fr) minmax(24rem, 0.9fr); gap: clamp(2rem, 6vw, 6rem); align-items: center; margin-top: 3rem; }
.music-stage__visual { position: relative; display: grid; min-height: 35rem; place-items: center; border: 1px solid #27272a; background: radial-gradient(circle, rgb(88 28 135 / 0.1), transparent 58%), rgb(9 9 11 / 0.62); }
.music-stage__visual::before, .music-stage__visual::after { position: absolute; width: 2rem; height: 2rem; border-color: #7c3aed; content: ''; }
.music-stage__visual::before { top: -1px; left: -1px; border-top: 1px solid; border-left: 1px solid; }
.music-stage__visual::after { right: -1px; bottom: -1px; border-right: 1px solid; border-bottom: 1px solid; }
.music-stage__caption { position: absolute; right: 1.5rem; bottom: 1.4rem; left: 1.5rem; display: flex; align-items: baseline; justify-content: space-between; gap: 1rem; border-top: 1px solid #27272a; padding-top: 1rem; }
.music-stage__caption span { color: #7c3aed; font-size: 0.58rem; letter-spacing: 0.22em; }
.music-stage__caption strong { min-width: 0; overflow: hidden; font-family: Georgia, serif; font-weight: 400; text-overflow: ellipsis; white-space: nowrap; }
.track-list { border-top: 1px solid #27272a; }
.track-list li { border-bottom: 1px solid #27272a; }
.track-card { display: grid; grid-template-columns: 2rem 4.5rem minmax(0, 1fr) auto 2.5rem; width: 100%; align-items: center; gap: 1rem; padding: 1rem 0.75rem; text-align: left; transition: background 180ms ease, padding 180ms ease; }
.track-card:hover, .track-card--current { background: linear-gradient(90deg, rgb(76 29 149 / 0.2), transparent); padding-right: 0.35rem; padding-left: 1.15rem; }
.track-card__number { color: #52525b; font-family: ui-monospace, monospace; font-size: 0.65rem; }
.track-card__cover { width: 4.5rem; aspect-ratio: 1; overflow: hidden; background: #18181b; }
.track-card__cover img { width: 100%; height: 100%; object-fit: cover; filter: saturate(0.7); transition: transform 350ms ease, filter 350ms ease; }
.track-card:hover img, .track-card--current img { filter: saturate(1); transform: scale(1.06); }
.track-card__copy { min-width: 0; }
.track-card__copy strong { display: block; overflow: hidden; font-family: Georgia, serif; font-size: 1.1rem; font-weight: 400; text-overflow: ellipsis; white-space: nowrap; }
.track-card__copy small { display: block; overflow: hidden; margin-top: 0.35rem; color: #52525b; font-size: 0.65rem; text-overflow: ellipsis; white-space: nowrap; }
.track-card__duration { color: #71717a; font-family: ui-monospace, monospace; font-size: 0.7rem; font-variant-numeric: tabular-nums; }
.track-card__action { display: grid; width: 2.25rem; aspect-ratio: 1; place-items: center; border: 1px solid #3f3f46; color: #c4b5fd; }
@media (max-width: 900px) { .music-stage { grid-template-columns: 1fr; } .music-stage__visual { min-height: 28rem; } }
@media (max-width: 640px) { .music-page__shell { padding-top: 2rem; } .music-page__header { display: block; } .music-page__summary { display: none; } .music-stage { margin-top: 2rem; } .music-stage__visual { min-height: 22rem; } .track-card { grid-template-columns: 1.8rem 3.5rem minmax(0, 1fr) 2.25rem; gap: 0.7rem; } .track-card__cover { width: 3.5rem; } .track-card__duration { display: none; } }
@media (prefers-reduced-motion: reduce) { .track-card, .track-card__cover img { transition: none; } }
</style>
