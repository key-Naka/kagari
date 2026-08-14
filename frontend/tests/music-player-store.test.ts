import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import { usePlayerStore, type PublicTrack } from '../stores/player'

const tracks: PublicTrack[] = [
  {
    id: 1,
    title: 'Night Archive',
    sortOrder: 10,
    cover: { id: 11, objectKey: 'cover-a', publicUrl: 'https://cdn.example.com/cover-a.webp', kind: 'image', mimeType: 'image/webp', size: 1000, originalName: 'cover-a.webp', width: 1200, height: 1200, durationMs: 0 },
    audio: { id: 12, objectKey: 'audio-a', publicUrl: 'https://cdn.example.com/audio-a.mp3', kind: 'audio', mimeType: 'audio/mpeg', size: 4000, originalName: 'audio-a.mp3', width: 0, height: 0, durationMs: 180000 },
  },
  {
    id: 2,
    title: 'First Light',
    sortOrder: 20,
    cover: { id: 21, objectKey: 'cover-b', publicUrl: 'https://cdn.example.com/cover-b.webp', kind: 'image', mimeType: 'image/webp', size: 1000, originalName: 'cover-b.webp', width: 1200, height: 1200, durationMs: 0 },
    audio: { id: 22, objectKey: 'audio-b', publicUrl: 'https://cdn.example.com/audio-b.ogg', kind: 'audio', mimeType: 'audio/ogg', size: 4000, originalName: 'audio-b.ogg', width: 0, height: 0, durationMs: 95000 },
  },
]

describe('Mini Player public state', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('hydrates Track data without autoplay and keeps playback selection across controls', () => {
    const player = usePlayerStore()

    player.hydrateTracks(tracks)
    expect(player.currentTrack?.id).toBe(1)
    expect(player.isPlaying).toBe(false)
    expect(player.hasStarted).toBe(false)

    player.playTrack(1)
    expect(player.isPlaying).toBe(true)
    expect(player.hasStarted).toBe(true)

    player.playNext()
    expect(player.currentTrack?.id).toBe(2)
    expect(player.isPlaying).toBe(true)

    player.pause()
    expect(player.isPlaying).toBe(false)

    player.playPrevious()
    expect(player.currentTrack?.id).toBe(1)
    expect(player.hasStarted).toBe(true)
  })

  it('moves to the first available Track when refreshed data no longer contains the selection', () => {
    const player = usePlayerStore()
    player.hydrateTracks(tracks)
    player.playTrack(2)

    player.hydrateTracks([tracks[0]!])

    expect(player.currentTrack?.id).toBe(1)
    expect(player.isPlaying).toBe(false)
  })
})
