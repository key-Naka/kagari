import { computed, shallowRef } from 'vue'
import { defineStore } from 'pinia'

export interface TrackMedia {
  id: number
  objectKey: string
  publicUrl: string
  kind: 'image' | 'audio'
  mimeType: string
  size: number
  originalName: string
  width: number
  height: number
  durationMs: number
}

export interface PublicTrack {
  id: number
  title: string
  cover: TrackMedia
  audio: TrackMedia
  sortOrder: number
}

const emptySpectrum = () => Array.from({ length: 32 }, () => 0)

export const usePlayerStore = defineStore('player', () => {
  const tracks = shallowRef<PublicTrack[]>([])
  const currentTrackId = shallowRef<number | null>(null)
  const isPlaying = shallowRef(false)
  const hasStarted = shallowRef(false)
  const currentTimeMs = shallowRef(0)
  const spectrum = shallowRef<number[]>(emptySpectrum())
  const errorMessage = shallowRef('')

  const currentTrack = computed(() => {
    return tracks.value.find(track => track.id === currentTrackId.value) ?? null
  })

  const durationMs = computed(() => currentTrack.value?.audio.durationMs ?? 0)

  function hydrateTracks(values: PublicTrack[]): void {
    tracks.value = [...values].sort((left, right) => left.sortOrder - right.sortOrder || left.id - right.id)
    if (currentTrackId.value === null) {
      currentTrackId.value = tracks.value[0]?.id ?? null
      return
    }
    if (!tracks.value.some(track => track.id === currentTrackId.value)) {
      currentTrackId.value = tracks.value[0]?.id ?? null
      isPlaying.value = false
      hasStarted.value = false
      currentTimeMs.value = 0
      spectrum.value = emptySpectrum()
    }
  }

  function playTrack(id: number): void {
    if (!tracks.value.some(track => track.id === id)) return
    if (currentTrackId.value !== id) currentTimeMs.value = 0
    currentTrackId.value = id
    hasStarted.value = true
    isPlaying.value = true
    errorMessage.value = ''
  }

  function togglePlayback(): void {
    if (currentTrackId.value === null && tracks.value.length > 0) {
      currentTrackId.value = tracks.value[0]?.id ?? null
    }
    if (currentTrackId.value === null) return
    hasStarted.value = true
    isPlaying.value = !isPlaying.value
    errorMessage.value = ''
  }

  function pause(): void {
    isPlaying.value = false
  }

  function move(direction: 1 | -1): void {
    if (tracks.value.length === 0) return
    const currentIndex = tracks.value.findIndex(track => track.id === currentTrackId.value)
    const start = currentIndex < 0 ? 0 : currentIndex
    const nextIndex = (start + direction + tracks.value.length) % tracks.value.length
    const nextTrack = tracks.value[nextIndex]
    if (!nextTrack) return
    currentTrackId.value = nextTrack.id
    currentTimeMs.value = 0
    hasStarted.value = true
    isPlaying.value = true
    errorMessage.value = ''
  }

  function playNext(): void {
    move(1)
  }

  function playPrevious(): void {
    move(-1)
  }

  function syncProgress(seconds: number): void {
    currentTimeMs.value = Math.max(0, Math.round(seconds * 1000))
  }

  function syncPlaying(value: boolean): void {
    isPlaying.value = value
  }

  function setSpectrum(values: Uint8Array | number[]): void {
    spectrum.value = Array.from(values)
  }

  function reportPlaybackError(message: string): void {
    isPlaying.value = false
    errorMessage.value = message
  }

  return {
    tracks,
    currentTrackId,
    currentTrack,
    durationMs,
    isPlaying,
    hasStarted,
    currentTimeMs,
    spectrum,
    errorMessage,
    hydrateTracks,
    playTrack,
    togglePlayback,
    pause,
    playNext,
    playPrevious,
    syncProgress,
    syncPlaying,
    setSpectrum,
    reportPlaybackError,
  }
})
