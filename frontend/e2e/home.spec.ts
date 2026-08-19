import { createServer, type Server } from 'node:http'
import { expect, test } from '@playwright/test'

type ApiMode = 'ready' | 'partial'

let apiServer: Server
let apiMode: ApiMode = 'ready'

const readyHomeResponse = {
  works: {
    availability: 'operational', count: 1,
    item: { title: 'Kagari Core', coverUrl: 'https://cdn.example.com/kagari.webp', description: '一套由 Nuxt 与 Go 共同驱动的公开档案系统。', featured: true },
  },
  blog: {
    availability: 'operational', count: 1,
    item: { title: 'Night Index', summary: '关于系统边界与夜间创作的最新记录。', publishedAt: '2026-08-19T08:00:00Z' },
  },
  music: {
    availability: 'operational', count: 1,
    item: { title: 'Ash Choir', coverUrl: 'https://cdn.example.com/cover.webp' },
  },
  github: {
    availability: 'operational', count: 1,
    item: { repository: 'key-Naka/kagari', description: 'Personal archive monolith' },
  },
  gallery: { availability: 'operational', count: 12 },
  status: { availability: 'operational', operational: 4, total: 4 },
  visitorMessages: {
    availability: 'operational', count: 1,
    item: { nickname: 'Aya', content: 'Hello from the edge' },
  },
}

const partialHomeResponse = {
  ...readyHomeResponse,
  blog: { availability: 'operational', count: 0 },
  music: { availability: 'unavailable', count: 0 },
  gallery: { availability: 'operational', count: 0 },
  visitorMessages: { availability: 'operational', count: 0 },
}

const readyResponses: Record<string, unknown> = {
  '/api/v1/home': readyHomeResponse,
  '/api/v1/projects': [{
    title: 'Kagari Core',
    slug: 'kagari-core',
    coverUrl: 'https://cdn.example.com/kagari.webp',
    description: '一套由 Nuxt 与 Go 共同驱动的公开档案系统。',
    technologies: ['Nuxt', 'Go'],
    types: ['Web'],
    featured: true,
    sortOrder: 0,
    websiteUrl: 'https://example.com',
    repositoryUrl: 'https://github.com/key-Naka/kagari',
  }],
  '/api/v1/posts': [{
    title: 'Night Index',
    slug: 'night-index',
    summary: '关于系统边界与夜间创作的最新记录。',
    tags: ['architecture'],
    publishedAt: '2026-08-19T08:00:00Z',
  }],
  '/api/v1/tracks': [{
    id: 1,
    title: 'Ash Choir',
    cover: { id: 1, objectKey: 'cover.webp', publicUrl: 'https://cdn.example.com/cover.webp', kind: 'image', mimeType: 'image/webp', size: 100, originalName: 'cover.webp', width: 800, height: 800 },
    audio: { id: 2, objectKey: 'track.mp3', publicUrl: 'https://cdn.example.com/track.mp3', kind: 'audio', mimeType: 'audio/mpeg', size: 1000, originalName: 'track.mp3', durationMs: 183000 },
    sortOrder: 0,
  }],
  '/api/v1/github': {
    availability: 'operational',
    contributions: [{ date: '2026-08-19', level: 3 }],
    activities: [{ kind: 'PushEvent', repository: 'key-Naka/kagari', occurredAt: '2026-08-19T08:00:00Z' }],
    repositories: [{ name: 'kagari', url: 'https://github.com/key-Naka/kagari', description: 'Personal archive monolith', language: 'Vue', stars: 13, updatedAt: '2026-08-19T08:00:00Z' }],
    sampledAt: '2026-08-19T09:00:00Z',
  },
  '/api/v1/service-status': {
    availability: 'operational',
    resources: { cpu: '18%', memory: '42%', disk: '31%', network: 'normal', uptime: '9d' },
    containers: [],
    applications: [
      { name: 'API', state: 'operational' },
      { name: 'HTTP', state: 'operational' },
      { name: 'MySQL', state: 'operational' },
      { name: 'Redis', state: 'operational' },
    ],
    sampledAt: '2026-08-19T09:00:00Z',
  },
  '/api/v1/visitor-messages': [{
    id: 1,
    nickname: 'Aya',
    content: 'Hello from the edge',
    createdAt: '2026-08-19T09:15:00Z',
  }],
}

test.beforeAll(async () => {
  apiServer = createServer((request, response) => {
    response.setHeader('Access-Control-Allow-Origin', 'http://127.0.0.1:3001')
    response.setHeader('Content-Type', 'application/json')

    const path = request.url ?? ''
    if (apiMode === 'partial' && path === '/api/v1/home') {
      response.writeHead(200)
      response.end(JSON.stringify(partialHomeResponse))
      return
    }

    const payload = readyResponses[path]
    if (payload === undefined) {
      response.writeHead(404)
      response.end('{"error":"not found"}')
      return
    }
    response.writeHead(200)
    response.end(JSON.stringify(payload))
  })

  await new Promise<void>((resolve, reject) => {
    apiServer.once('error', reject)
    apiServer.listen(8080, '127.0.0.1', resolve)
  })
})

test.afterAll(async () => {
  await new Promise<void>((resolve, reject) => apiServer.close(error => error ? reject(error) : resolve()))
})

test('home presents API-backed highlights and every independent archive entry', async ({ page }) => {
  apiMode = 'ready'
  await page.goto('/')

  await expect(page.getByRole('heading', { name: /把系统、界面与声音，\s*归入同一座档案。/ })).toBeVisible()
  await expect(page.getByText('全栈工程师 / 独立创作者')).toBeVisible()
  await expect(page.getByText('Kagari Core')).toBeVisible()
  await expect(page.getByText('Night Index')).toBeVisible()
  await expect(page.getByText('Ash Choir')).toBeVisible()
  await expect(page.getByText('key-Naka/kagari', { exact: true })).toBeVisible()
  await expect(page.getByText('4 / 4 项公开检查正常')).toBeVisible()
  await expect(page.getByText('Hello from the edge')).toBeVisible()
  await expect(page.getByText('12 个 Album Item')).toBeVisible()

  for (const label of ['作品', '博客', '音乐', 'GitHub', '相册', '服务状态', '访客留言']) {
    await expect(page.getByRole('link', { name: `进入${label}档案` })).toBeVisible()
  }

})

for (const [label, route] of [
  ['作品', '/works'],
  ['博客', '/blog'],
  ['音乐', '/music'],
  ['GitHub', '/github'],
  ['相册', '/gallery'],
  ['服务状态', '/status'],
  ['访客留言', '/visitor-messages'],
] as const) {
  test(`home routes the ${label} entry to ${route}`, async ({ page }) => {
    apiMode = 'ready'
    await page.goto('/')
    await page.getByRole('link', { name: `进入${label}档案` }).click()
    await expect(page).toHaveURL(new RegExp(`${route}$`))
  })
}

test('home keeps the other archives usable when one API fails and others are empty', async ({ page }) => {
  apiMode = 'partial'
  await page.goto('/')

  await expect(page.getByText('Kagari Core')).toBeVisible()
  await expect(page.getByText('音乐摘要暂时离线')).toBeVisible()
  await expect(page.getByText('尚无公开文章')).toBeVisible()
  await expect(page.getByText('等待第一张 Album Item')).toBeVisible()
  await expect(page.getByText('等待第一条 Visitor Message')).toBeVisible()
  await expect(page.getByRole('link', { name: '进入音乐档案' })).toHaveAttribute('href', '/music')
  await expect(page.getByRole('link', { name: '进入博客档案' })).toHaveAttribute('href', '/blog')

  await page.route('**/api/v1/home', async route => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      ...partialHomeResponse,
      gallery: { availability: 'unavailable', count: 0 },
    }),
  }))
  await page.getByRole('button', { name: '同步摘要' }).click()
  await expect(page.getByText('相册摘要暂时离线')).toBeVisible()
})
