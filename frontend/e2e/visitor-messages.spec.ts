import { createServer, type IncomingMessage, type Server } from 'node:http'
import { expect, test } from '@playwright/test'

interface VisitorMessagePayload {
  nickname: string
  email: string
  content: string
}

let apiServer: Server
let submittedPayload: VisitorMessagePayload | null
let messages: Array<{ id: number, nickname: string, email?: string, content: string, createdAt: string }>

async function requestBody(request: IncomingMessage): Promise<VisitorMessagePayload> {
  const chunks: Buffer[] = []
  for await (const chunk of request) chunks.push(Buffer.from(chunk))
  return JSON.parse(Buffer.concat(chunks).toString('utf8')) as VisitorMessagePayload
}

test.beforeAll(async () => {
  apiServer = createServer(async (request, response) => {
    response.setHeader('Access-Control-Allow-Origin', 'http://127.0.0.1:3001')
    response.setHeader('Access-Control-Allow-Credentials', 'true')
    response.setHeader('Access-Control-Allow-Headers', 'Content-Type')
    response.setHeader('Access-Control-Allow-Methods', 'GET,POST,DELETE,OPTIONS')
    if (request.method === 'OPTIONS') {
      response.writeHead(204)
      response.end()
      return
    }
    if (request.url === '/api/v1/admin/session') {
      response.writeHead(200, { 'Content-Type': 'application/json' })
      response.end('{"authenticated":true}')
      return
    }
    if (request.url === '/api/v1/visitor-messages' && request.method === 'GET') {
      response.writeHead(200, { 'Content-Type': 'application/json' })
      response.end(JSON.stringify(messages.map(({ email: _email, ...message }) => message)))
      return
    }
    if (request.url === '/api/v1/visitor-messages' && request.method === 'POST') {
      submittedPayload = await requestBody(request)
      if (submittedPayload.content === 'RATE_LIMIT') {
        response.writeHead(429, { 'Content-Type': 'application/json', 'Retry-After': '60' })
        response.end('{"error":"visitor message rate limit exceeded"}')
        return
      }
      const created = {
        id: 2,
        nickname: submittedPayload.nickname,
        email: submittedPayload.email,
        content: submittedPayload.content,
        createdAt: '2026-08-19T09:30:00Z',
      }
      messages.unshift(created)
      const { email: _email, ...publicMessage } = created
      response.writeHead(201, { 'Content-Type': 'application/json' })
      response.end(JSON.stringify(publicMessage))
      return
    }
    if (request.url === '/api/v1/admin/visitor-messages' && request.method === 'GET') {
      response.writeHead(200, { 'Content-Type': 'application/json' })
      response.end(JSON.stringify(messages))
      return
    }
    if (request.url === '/api/v1/admin/visitor-messages/1' && request.method === 'DELETE') {
      messages = messages.filter(message => message.id !== 1)
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
  submittedPayload = null
  messages = [{
    id: 1,
    nickname: '',
    email: 'private@example.com',
    content: 'Existing public signal',
    createdAt: '2026-08-19T08:00:00Z',
  }]
})

test('visitor submits a message that appears immediately without exposing email', async ({ page }) => {
  await page.goto('/visitor-messages')
  await expect(page.getByRole('heading', { name: '访客留言' })).toBeVisible()
  await expect(page.getByText('Existing public signal')).toBeVisible()
  await expect(page.getByText('private@example.com')).toHaveCount(0)

  await page.locator('input[name="nickname"]').fill('Aya')
  await page.locator('input[name="email"]').fill('aya@example.com')
  await page.locator('textarea[name="content"]').fill('A new public signal')
  await page.getByRole('button', { name: '发送讯号' }).click()

  await expect(page.getByRole('status')).toContainText('讯号已进入公开档案')
  await expect(page.getByText('A new public signal')).toBeVisible()
  await expect(page.getByText('aya@example.com')).toHaveCount(0)
  expect(submittedPayload).toEqual({ nickname: 'Aya', email: 'aya@example.com', content: 'A new public signal' })
})

test('visitor sees an actionable rate-limit error', async ({ page }) => {
  await page.goto('/visitor-messages')
  await page.locator('textarea[name="content"]').fill('RATE_LIMIT')
  await page.getByRole('button', { name: '发送讯号' }).click()
  await expect(page.getByRole('alert')).toContainText('发送过于频繁，请稍后再试')
})

test('administrator can see the private email and permanently delete the message', async ({ page }) => {
  page.on('dialog', dialog => dialog.accept())
  await page.goto('/admin/visitor-messages')
  await expect(page.getByRole('heading', { name: '访客留言管理' })).toBeVisible()
  await expect(page.getByText('private@example.com')).toBeVisible()
  await page.getByRole('button', { name: '永久删除' }).click()
  await expect(page.getByText('Visitor Message 已永久删除')).toBeVisible()
  await expect(page.getByText('Existing public signal')).toHaveCount(0)
})
