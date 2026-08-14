import { createServer, type Server } from 'node:http'
import { expect, test } from '@playwright/test'

const silentAudioUrl = 'data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQAAAAA='

const tracks = [
  {
    id: 1,
    title: 'Night Archive',
    sortOrder: 10,
    cover: { id: 11, objectKey: 'media/image/2026/08/cover-a.webp', publicUrl: 'https://cdn.example.com/cover-a.webp', kind: 'image', mimeType: 'image/webp', size: 1000, originalName: 'cover-a.webp', width: 1200, height: 1200 },
    audio: { id: 12, objectKey: 'media/audio/2026/08/audio-a.mp3', publicUrl: silentAudioUrl, kind: 'audio', mimeType: 'audio/mpeg', size: 4000, originalName: 'audio-a.mp3', durationMs: 180000 },
  },
  {
    id: 2,
    title: 'First Light',
    sortOrder: 20,
    cover: { id: 21, objectKey: 'media/image/2026/08/cover-b.webp', publicUrl: 'https://cdn.example.com/cover-b.webp', kind: 'image', mimeType: 'image/webp', size: 1000, originalName: 'cover-b.webp', width: 1200, height: 1200 },
    audio: { id: 22, objectKey: 'media/audio/2026/08/audio-b.ogg', publicUrl: silentAudioUrl, kind: 'audio', mimeType: 'audio/ogg', size: 4000, originalName: 'audio-b.ogg', durationMs: 95000 },
  },
]

let apiServer: Server

test.beforeAll(async () => {
  apiServer = createServer((request, response) => {
    if (request.url === '/api/v1/tracks') {
      response.writeHead(200, { 'Content-Type': 'application/json' })
      response.end(JSON.stringify(tracks))
      return
    }
    response.writeHead(404)
    response.end()
  })
  await new Promise<void>((resolve, reject) => {
    apiServer.once('error', reject)
    apiServer.listen(8080, '127.0.0.1', resolve)
  })
})

test.afterAll(async () => {
  await new Promise<void>((resolve, reject) => apiServer.close(error => error ? reject(error) : resolve()))
})

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    const controls = window as unknown as {
      __audioPlayCalls: number
      __deferAudioResume: boolean
      __resolveAudioResume: () => void
    }
    const pendingResumes: Array<() => void> = []
    controls.__audioPlayCalls = 0
    controls.__deferAudioResume = false
    controls.__resolveAudioResume = () => pendingResumes.splice(0).forEach(resolve => resolve())
    class FakeAnalyser {
      fftSize = 64
      frequencyBinCount = 32
      connect() {}
      getByteFrequencyData(values: Uint8Array) { values.fill(144) }
    }
    class FakeAudioContext {
      state = 'suspended'
      destination = {}
      createMediaElementSource() { return { connect() {} } }
      createAnalyser() { return new FakeAnalyser() }
      resume() {
        if (!controls.__deferAudioResume) {
          this.state = 'running'
          return Promise.resolve()
        }
        return new Promise<void>(resolve => pendingResumes.push(() => {
          this.state = 'running'
          resolve()
        }))
      }
      close() { return Promise.resolve() }
    }
    Object.defineProperty(window, 'AudioContext', { configurable: true, value: FakeAudioContext })
    HTMLMediaElement.prototype.play = function () {
      controls.__audioPlayCalls++
      this.dispatchEvent(new Event('play'))
      return Promise.resolve()
    }
    HTMLMediaElement.prototype.pause = function () {
      this.dispatchEvent(new Event('pause'))
    }
    HTMLMediaElement.prototype.load = function () {}
  })
})

test('visitor starts playback and keeps control after leaving the music page', async ({ page }, testInfo) => {
  await page.goto('/music')

  await expect(page.getByRole('heading', { name: '音乐档案' })).toBeVisible()
  await expect(page.getByTestId('mini-player')).toBeHidden()
  await expect(page.getByRole('button', { name: '播放 Night Archive' })).toBeVisible()

  await page.getByRole('button', { name: '播放 Night Archive' }).click()
  await expect(page.getByTestId('music-visualizer')).toHaveAttribute('data-active', 'true')
  if (process.env.CAPTURE_MUSIC_VISUALS === '1') {
    await page.screenshot({ path: testInfo.outputPath('music-page.png'), fullPage: true })
  }

  await Promise.all([
    page.waitForURL('**/'),
    page.locator('main').getByRole('link', { name: '返回首页' }).click(),
  ])
  await Promise.all([
    page.waitForURL('**/works'),
    page.locator('main').getByRole('link', { name: '作品' }).click(),
  ])
  const miniPlayer = page.getByTestId('mini-player')
  await expect(miniPlayer).toBeVisible()
  await expect(miniPlayer).toContainText('Night Archive')

  await miniPlayer.getByRole('button', { name: '暂停' }).click()
  await expect(miniPlayer.getByRole('button', { name: '继续播放' })).toBeVisible()

  await miniPlayer.getByRole('button', { name: '下一首' }).click()
  await expect(miniPlayer).toContainText('First Light')
  if (process.env.CAPTURE_MUSIC_VISUALS === '1') {
    await page.screenshot({ path: testInfo.outputPath('mini-player.png'), fullPage: true })
  }
  await page.locator('audio').dispatchEvent('error')
  await expect(miniPlayer.getByRole('alert')).toContainText('音频加载失败')
})

test('a pending AudioContext resume cannot restart playback after pause', async ({ page }) => {
  await page.goto('/music')
  await expect(page.getByRole('button', { name: '播放 Night Archive' })).toBeVisible()
  await page.evaluate(() => {
    (window as unknown as { __deferAudioResume: boolean }).__deferAudioResume = true
  })

  await page.getByRole('button', { name: '播放 Night Archive' }).click()
  await page.getByRole('button', { name: '暂停 Night Archive' }).click()
  await page.evaluate(() => {
    (window as unknown as { __resolveAudioResume: () => void }).__resolveAudioResume()
  })

  await expect(page.getByTestId('music-visualizer')).toHaveAttribute('data-active', 'false')
  await expect.poll(() => page.evaluate(() => (window as unknown as { __audioPlayCalls: number }).__audioPlayCalls)).toBe(0)
})
