import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

async function readFrontendFile(path: string): Promise<string> {
  return readFile(resolve(process.cwd(), path), 'utf8')
}

describe('Track 页面与管理契约', () => {
  it('公开音乐页从 SSR API 载入 Track，并由常驻音频组件提供 Web Audio 与 Mini Player', async () => {
    const page = await readFrontendFile('pages/music/index.vue')
    const app = await readFrontendFile('app.vue')
    const player = await readFrontendFile('components/GlobalAudioPlayer.client.vue')

    expect(page).toContain('useAsyncData')
    expect(page).toContain('/api/v1/tracks')
    expect(page).toContain('player.hydrateTracks')
    expect(page).toContain("typeof value.width === 'number' ? value.width : 0")
    expect(page).toContain("typeof value.durationMs === 'number' ? value.durationMs : 0")
    expect(page).toContain('<MusicVisualizer')
    expect(app).toContain('<GlobalAudioPlayer')
    expect(player).toContain('new AudioContext()')
    expect(player).toContain('createAnalyser()')
    expect(player).toContain('data-testid="mini-player"')
    expect(player).toContain("route.path === '/music'")
    expect(player).toContain("matchMedia('(prefers-reduced-motion: reduce)')")
    expect(player).toContain('playbackOperation')
    expect(player).toContain('@error="handlePlaybackError"')
    expect(player).not.toContain('autoplay')
  })

  it('管理员页面自动识别媒体元数据，通过受保护接口创建、编辑、启停和排序 Track', async () => {
    const page = await readFrontendFile('pages/admin/tracks.vue')
    const dashboard = await readFrontendFile('pages/admin/index.vue')
    const mediaUpload = await readFrontendFile('composables/useAdminMediaUpload.ts')

    expect(page).toContain("definePageMeta({ middleware: 'admin-auth' })")
    expect(page).toContain('useAdminMediaUpload()')
    expect(mediaUpload).toContain('/api/v1/admin/media/upload-credentials')
    expect(mediaUpload).toContain('/api/v1/admin/media')
    expect(page).toContain('/api/v1/admin/tracks')
    expect(page).toContain("method: activeTrackId.value === null ? 'POST' : 'PUT'")
    expect(mediaUpload).toContain("audio.addEventListener('loadedmetadata'")
    expect(page).toContain('onMounted(loadTracks)')
    expect(mediaUpload).toContain('durationMs: Math.round(audio.duration * 1000)')
    expect(page).toContain('v-model="form.enabled"')
    expect(page).toContain('v-model.number="form.sortOrder"')
    expect(dashboard).toContain('to="/admin/tracks"')
  })
})
