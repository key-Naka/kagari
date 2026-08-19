import { createServer, type IncomingMessage, type Server, type ServerResponse } from 'node:http'
import { expect, test } from '@playwright/test'

interface AlbumItemPayload {
  title: string
  note: string
  alt: string
  year: string
  imageMediaId: number
  anchorX: number
  anchorY: number
  width: string
  aspectRatio: string
  colors: [string, string, string]
  published: boolean
  sortOrder: number
}

let apiServer: Server
let authenticated = true
let nextMediaId = 1
let nextAlbumItemId = 1
let siteConfig = { siteTitle: 'Kagari', seoSummary: '公开档案', shareImageUrl: '' }
let projects: unknown[]
let posts: unknown[]
let tracks: unknown[]
let mediaCatalog: unknown[]
let requestedMediaKind: 'image' | 'audio' = 'image'
let uploadBody = ''
let albumItems: Array<AlbumItemPayload & {
  id: number
  image: {
    id: number
    objectKey: string
    publicUrl: string
    kind: 'image'
    mimeType: string
    size: number
    originalName: string
    width: number
    height: number
    createdAt: string
  }
}>

async function jsonBody<T>(request: IncomingMessage): Promise<T> {
  const chunks: Buffer[] = []
  for await (const chunk of request) chunks.push(Buffer.from(chunk))
  return JSON.parse(Buffer.concat(chunks).toString('utf8')) as T
}

function json(response: ServerResponse, status: number, body: unknown): void {
  response.writeHead(status, { 'Content-Type': 'application/json' })
  response.end(JSON.stringify(body))
}

function oneSecondWav(): Buffer {
  const sampleRate = 8000
  const dataSize = sampleRate * 2
  const buffer = Buffer.alloc(44 + dataSize)
  buffer.write('RIFF', 0)
  buffer.writeUInt32LE(36 + dataSize, 4)
  buffer.write('WAVEfmt ', 8)
  buffer.writeUInt32LE(16, 16)
  buffer.writeUInt16LE(1, 20)
  buffer.writeUInt16LE(1, 22)
  buffer.writeUInt32LE(sampleRate, 24)
  buffer.writeUInt32LE(sampleRate * 2, 28)
  buffer.writeUInt16LE(2, 32)
  buffer.writeUInt16LE(16, 34)
  buffer.write('data', 36)
  buffer.writeUInt32LE(dataSize, 40)
  return buffer
}

test.beforeAll(async () => {
  apiServer = createServer(async (request, response) => {
    response.setHeader('Access-Control-Allow-Origin', 'http://127.0.0.1:3001')
    response.setHeader('Access-Control-Allow-Credentials', 'true')
    response.setHeader('Access-Control-Allow-Headers', 'Content-Type')
    response.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,DELETE,OPTIONS')
    if (request.method === 'OPTIONS') {
      response.writeHead(204)
      response.end()
      return
    }
    if (request.url === '/api/v1/admin/session' && request.method === 'GET') {
      json(response, authenticated ? 200 : 401, authenticated ? { authenticated: true } : { error: 'unauthorized' })
      return
    }
    if (!authenticated) {
      json(response, 401, { error: 'unauthorized' })
      return
    }
    if (request.url === '/api/v1/admin/site-config' && request.method === 'GET') {
      json(response, 200, siteConfig)
      return
    }
    if (request.url === '/api/v1/admin/site-config' && request.method === 'PUT') {
      siteConfig = await jsonBody<typeof siteConfig>(request)
      json(response, 200, siteConfig)
      return
    }
    for (const [path, collection] of [
      ['/api/v1/admin/projects', projects],
      ['/api/v1/admin/posts', posts],
      ['/api/v1/admin/tracks', tracks],
    ] as const) {
      if (request.url === path && request.method === 'GET') {
        json(response, 200, collection)
        return
      }
      if (request.url === path && request.method === 'POST') {
        const payload = await jsonBody<Record<string, unknown>>(request)
        const catalog = mediaCatalog as Array<Record<string, unknown>>
        const created = path.endsWith('/tracks')
          ? {
              ...payload,
              id: collection.length + 1,
              cover: catalog.find(item => item.id === payload.coverMediaId),
              audio: catalog.find(item => item.id === payload.audioMediaId),
            }
          : { ...payload, id: collection.length + 1 }
        collection.push(created)
        json(response, 201, created)
        return
      }
    }
    if (request.url === '/api/v1/admin/media' && request.method === 'GET') {
      json(response, 200, mediaCatalog)
      return
    }
    if (request.url === '/api/v1/admin/gallery-items' && request.method === 'GET') {
      json(response, 200, albumItems)
      return
    }
    if (request.url === '/api/v1/admin/media/upload-credentials' && request.method === 'POST') {
      const payload = await jsonBody<{ kind: 'image' | 'audio', filename: string }>(request)
      requestedMediaKind = payload.kind
      json(response, 200, {
        uploadToken: 'short-lived-token',
        uploadUrl: 'http://127.0.0.1:8080/qiniu-upload',
        objectKey: `media/${payload.kind}/2026/08/${payload.filename}`,
      })
      return
    }
    if (request.url === '/qiniu-upload' && request.method === 'POST') {
      const chunks: Buffer[] = []
      for await (const chunk of request) chunks.push(Buffer.from(chunk))
      uploadBody = Buffer.concat(chunks).toString('latin1')
      json(response, 200, { key: 'uploaded' })
      return
    }
    if (request.url === '/api/v1/admin/media' && request.method === 'POST') {
      const payload = await jsonBody<{
        objectKey: string
        kind: 'image' | 'audio'
        mimeType: string
        size: number
        originalName: string
        width: number
        height?: number
        durationMs?: number
      }>(request)
      const created = {
        id: nextMediaId++,
        objectKey: payload.objectKey,
        publicUrl: payload.kind === 'image' ? 'https://cdn.example.com/admin-upload.png' : 'https://cdn.example.com/admin-upload.wav',
        kind: payload.kind,
        mimeType: payload.mimeType,
        size: payload.size,
        originalName: payload.originalName,
        width: payload.width,
        height: payload.height,
        durationMs: payload.durationMs,
        createdAt: '2026-08-19T10:00:00Z',
      }
      mediaCatalog.unshift(created)
      json(response, 201, created)
      return
    }
    if (request.url === '/api/v1/admin/gallery-items' && request.method === 'POST') {
      const payload = await jsonBody<AlbumItemPayload>(request)
      const created = {
        ...payload,
        id: nextAlbumItemId++,
        image: {
          id: payload.imageMediaId,
          objectKey: 'media/image/2026/08/admin-upload.png',
          publicUrl: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Z8icAAAAASUVORK5CYII=',
          kind: 'image' as const,
          mimeType: 'image/png',
          size: 68,
          originalName: 'album.png',
          width: 1,
          height: 1,
          createdAt: '2026-08-19T10:00:00Z',
        },
      }
      albumItems.push(created)
      json(response, 201, created)
      return
    }
    const deleteMatch = request.url?.match(/^\/api\/v1\/admin\/gallery-items\/(\d+)$/)
    if (deleteMatch && request.method === 'DELETE') {
      albumItems = albumItems.filter(item => item.id !== Number(deleteMatch[1]))
      response.writeHead(204)
      response.end()
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

test.beforeEach(() => {
  authenticated = true
  nextMediaId = 1
  nextAlbumItemId = 1
  albumItems = []
  siteConfig = { siteTitle: 'Kagari', seoSummary: '公开档案', shareImageUrl: '' }
  projects = []
  posts = []
  tracks = []
  mediaCatalog = []
  requestedMediaKind = 'image'
  uploadBody = ''
})

test('未认证访问 Administration Console 会跳转登录页', async ({ page }) => {
  authenticated = false
  const sessionCheck = page.waitForResponse(response => response.url().endsWith('/api/v1/admin/session'))
  await page.goto('/admin')
  expect((await sessionCheck).status()).toBe(401)
  await expect(page).toHaveURL(/\/admin\/login$/)
  await expect(page.getByRole('heading', { name: '管理员登录' })).toBeVisible()
})

test('管理员从统一控制台上传、发布并永久删除 Album Item', async ({ page }) => {
  await page.goto('/admin')
  const galleryLink = page.getByRole('link', { name: '管理 Album Item' })
  await expect(galleryLink).toBeVisible()
  await galleryLink.click()

  await expect(page).toHaveURL(/\/admin\/gallery-items$/)
  await expect(page.getByRole('heading', { name: 'Album Item 管理' })).toBeVisible()
  await expect(page.getByText('还没有 Album Item')).toBeVisible()

  await page.getByLabel('标题').fill('Violet Wake')
  await page.getByLabel('年份').fill('2026')
  await page.getByLabel('档案注记').fill('afterimage / 00:14')
  await page.getByLabel('图片替代文本').fill('紫色光晕穿过深色轨道')
  await page.getByLabel('公开发布').check()
  await page.getByLabel('图片文件').setInputFiles({
    name: 'album.png',
    mimeType: 'image/png',
    buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Z8icAAAAASUVORK5CYII=', 'base64'),
  })
  await page.getByRole('button', { name: '创建 Album Item' }).click()

  await expect(page.getByRole('status')).toContainText('Album Item 已创建')
  await expect(page.getByLabel('图片文件')).toHaveValue('')
  await expect(page.getByRole('heading', { name: 'Violet Wake' })).toBeVisible()
  await expect(page.getByText('公开', { exact: true })).toBeVisible()
  expect(uploadBody).toContain('short-lived-token')
  expect(uploadBody).toContain('media/image/2026/08/album.png')

  page.on('dialog', dialog => dialog.accept())
  await page.getByRole('button', { name: '永久删除' }).click()
  await expect(page.getByRole('status')).toContainText('Album Item 已永久删除')
  await expect(page.getByText('还没有 Album Item')).toBeVisible()
})

test('管理员完成作品、文章与站点配置关键操作', async ({ page }) => {
  await page.goto('/admin/projects')
  await page.getByLabel('标题').fill('Archive Engine')
  await page.getByLabel('稳定 slug').fill('archive-engine')
  await page.getByLabel('封面 HTTPS 地址').fill('https://cdn.example.com/cover.png')
  await page.getByLabel('说明').fill('面向公开档案的系统。')
  await page.getByLabel('状态').selectOption('published')
  await page.getByRole('button', { name: '创建作品' }).click()
  await expect(page.getByRole('status')).toContainText('作品已发布')
  await expect(page.getByRole('heading', { name: 'Archive Engine' })).toBeVisible()

  await page.goto('/admin/posts')
  await page.getByLabel('标题').fill('Building the Archive')
  await page.getByLabel('稳定 slug').fill('building-the-archive')
  await page.getByLabel('摘要').fill('从边界到界面的记录。')
  await page.getByLabel('分享封面 HTTPS 地址').fill('https://cdn.example.com/building-the-archive.webp')
  await page.getByLabel('Markdown 内容').fill('# Archive')
  await page.getByLabel('状态').selectOption('published')
  await page.getByRole('button', { name: '创建文章' }).click()
  await expect(page.getByRole('status')).toContainText('文章已创建')
  expect(posts).toContainEqual(expect.objectContaining({ shareImageUrl: 'https://cdn.example.com/building-the-archive.webp' }))

  await page.goto('/admin')
  await page.getByLabel('站点标题').fill('Kagari Archive')
  await page.getByLabel('SEO 摘要').fill('系统、界面、文字与声音的公开档案。')
  await page.getByLabel('分享图片 HTTPS 地址').fill('https://cdn.example.com/share.png')
  await page.getByRole('button', { name: '保存配置' }).click()
  await expect(page.getByRole('status')).toContainText('站点配置已保存')
  expect(siteConfig.siteTitle).toBe('Kagari Archive')
})

test('管理员上传封面和音频并创建 Track', async ({ page }) => {
  await page.goto('/admin/tracks')
  await expect(page.getByText('还没有 Track')).toBeVisible()
  await page.getByLabel('标题').fill('Signal Bloom')
  await page.getByLabel('公开启用').check()
  await page.getByLabel('封面文件').setInputFiles({
    name: 'cover.png',
    mimeType: 'image/png',
    buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Z8icAAAAASUVORK5CYII=', 'base64'),
  })
  await page.getByLabel('音频文件').setInputFiles({ name: 'signal.wav', mimeType: 'audio/wav', buffer: oneSecondWav() })
  await page.getByRole('button', { name: '创建 Track' }).click()
  await expect(page.getByRole('status')).toContainText('Track 已创建')
  await expect(page.getByText('Signal Bloom')).toBeVisible()
  expect(requestedMediaKind).toBe('audio')
  expect(mediaCatalog).toHaveLength(2)
})
