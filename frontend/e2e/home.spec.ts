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
  '/api/v1/site-config': {
    siteTitle: 'Kagari · 全栈工程师与独立创作者',
    seoSummary: 'Kagari 的作品导向首页：浏览全栈工程、Blog Post、Track、GitHub、相册、服务状态与 Visitor Message 档案。',
    shareImageUrl: '',
  },
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
  '/api/v1/projects/kagari-core': {
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
  },
  '/api/v1/posts': [{
    title: 'Night Index',
    slug: 'night-index',
    summary: '关于系统边界与夜间创作的最新记录。',
    shareImageUrl: 'https://cdn.example.com/night-index.webp',
    tags: ['architecture'],
    publishedAt: '2026-08-19T08:00:00Z',
  }],
  '/api/v1/posts/night-index': {
    title: 'Night Index',
    slug: 'night-index',
    summary: '关于系统边界与夜间创作的最新记录。',
    shareImageUrl: 'https://cdn.example.com/night-index.webp',
    tags: ['architecture'],
    publishedAt: '2026-08-19T08:00:00Z',
    content: '<p>夜间档案正文。</p>',
  },
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

test('home SSR document exposes complete Chinese search and share metadata', async ({ page }) => {
  apiMode = 'ready'
  await page.goto('/')

  await expect(page).toHaveTitle('Kagari · 全栈工程师与独立创作者')
  await expect(page.locator('html')).toHaveAttribute('lang', 'zh-CN')
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', 'https://ykagari.top/')
  await expect(page.locator('meta[name="description"]')).toHaveAttribute('content', /Kagari 的作品导向首页/)
  await expect(page.locator('meta[property="og:title"]')).toHaveAttribute('content', 'Kagari · 全栈工程师与独立创作者')
  await expect(page.locator('meta[property="og:description"]')).toHaveAttribute('content', /Kagari 的作品导向首页/)
  await expect(page.locator('meta[property="og:url"]')).toHaveAttribute('content', 'https://ykagari.top/')
  await expect(page.locator('meta[property="og:type"]')).toHaveAttribute('content', 'website')
  await expect(page.locator('meta[property="og:image"]')).toHaveAttribute('content', 'https://ykagari.top/share-card.png')
  await expect(page.locator('meta[name="twitter:card"]')).toHaveAttribute('content', 'summary_large_image')
  await expect(page.locator('meta[name="twitter:title"]')).toHaveAttribute('content', 'Kagari · 全栈工程师与独立创作者')
  await expect(page.locator('meta[name="twitter:description"]')).toHaveAttribute('content', /Kagari 的作品导向首页/)
  await expect(page.locator('meta[name="twitter:image"]')).toHaveAttribute('content', 'https://ykagari.top/share-card.png')
})

test('content metadata is present in the SSR response before hydration', async ({ request }) => {
  const response = await request.get('/works/kagari-core')
  expect(response.ok()).toBe(true)
  const html = await response.text()

  expect(html).toMatch(/<html\s+lang="zh-CN"/)
  expect(html).toContain('<title>Kagari Core · 作品 · Kagari</title>')
  expect(html).toContain('name="description" content="一套由 Nuxt 与 Go 共同驱动的公开档案系统。"')
  expect(html).toContain('rel="canonical" href="https://ykagari.top/works/kagari-core"')
  expect(html).toContain('property="og:image" content="https://cdn.example.com/kagari.webp"')
  expect(html).toContain('name="twitter:card" content="summary_large_image"')
})

test('every independent public module exposes complete route-specific metadata', async ({ page }) => {
  apiMode = 'ready'
  const routes = [
    ['/works', '作品档案 · Kagari'],
    ['/blog', '写作档案 · Kagari'],
    ['/music', '音乐档案 · Kagari'],
    ['/github', 'GitHub 档案 · Kagari'],
    ['/gallery', '无界相册 · Kagari'],
    ['/status', '服务状态 · Kagari'],
    ['/visitor-messages', '访客留言 · Kagari'],
  ] as const

  for (const [route, title] of routes) {
    await page.goto(route)
    await expect(page).toHaveTitle(title)
    await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', `https://ykagari.top${route}`)
    await expect(page.locator('meta[name="description"]')).toHaveAttribute('content', /[\u3400-\u9fff]/)
    await expect(page.locator('meta[property="og:title"]')).toHaveAttribute('content', title)
    await expect(page.locator('meta[property="og:url"]')).toHaveAttribute('content', `https://ykagari.top${route}`)
    await expect(page.locator('meta[property="og:image"]')).toHaveAttribute('content', 'https://ykagari.top/share-card.png')
    await expect(page.locator('meta[name="twitter:title"]')).toHaveAttribute('content', title)
    await expect(page.locator('meta[name="twitter:description"]')).toHaveAttribute('content', /[\u3400-\u9fff]/)
    await expect(page.locator('meta[name="twitter:image"]')).toHaveAttribute('content', 'https://ykagari.top/share-card.png')
  }
})

test('Portfolio Project detail uses its summary and cover for sharing metadata', async ({ page }) => {
  apiMode = 'ready'
  await page.goto('/works/kagari-core')

  await expect(page).toHaveTitle('Kagari Core · 作品 · Kagari')
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', 'https://ykagari.top/works/kagari-core')
  await expect(page.locator('meta[name="description"]')).toHaveAttribute('content', '一套由 Nuxt 与 Go 共同驱动的公开档案系统。')
  await expect(page.locator('meta[property="og:type"]')).toHaveAttribute('content', 'article')
  await expect(page.locator('meta[property="og:image"]')).toHaveAttribute('content', 'https://cdn.example.com/kagari.webp')
  await expect(page.locator('meta[name="twitter:image"]')).toHaveAttribute('content', 'https://cdn.example.com/kagari.webp')
})

test('Blog Post detail uses its summary and configured sharing cover', async ({ page }) => {
  await page.goto('/blog/night-index')

  await expect(page).toHaveTitle('Night Index · 博客 · Kagari')
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', 'https://ykagari.top/blog/night-index')
  await expect(page.locator('meta[name="description"]')).toHaveAttribute('content', '关于系统边界与夜间创作的最新记录。')
  await expect(page.locator('meta[property="og:type"]')).toHaveAttribute('content', 'article')
  await expect(page.locator('meta[property="og:image"]')).toHaveAttribute('content', 'https://cdn.example.com/night-index.webp')
  await expect(page.locator('meta[name="twitter:image"]')).toHaveAttribute('content', 'https://cdn.example.com/night-index.webp')
})

test('robots and sitemap expose indexable public routes without administrator pages', async ({ request }) => {
  const robotsResponse = await request.get('/robots.txt')
  expect(robotsResponse.ok()).toBe(true)
  expect(robotsResponse.headers()['content-type']).toContain('text/plain')
  const robots = await robotsResponse.text()
  expect(robots).toContain('User-agent: *')
  expect(robots).toContain('Allow: /')
  expect(robots).toContain('Disallow: /admin')
  expect(robots).toContain('Sitemap: https://ykagari.top/sitemap.xml')

  const sitemapResponse = await request.get('/sitemap.xml')
  expect(sitemapResponse.ok()).toBe(true)
  expect(sitemapResponse.headers()['content-type']).toContain('application/xml')
  const sitemap = await sitemapResponse.text()
  for (const location of ['/', '/works', '/works/kagari-core', '/blog', '/blog/night-index', '/music', '/github', '/gallery', '/status', '/visitor-messages']) {
    expect(sitemap).toContain(`<loc>https://ykagari.top${location}</loc>`)
  }
  expect(sitemap).not.toContain('/admin')
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
