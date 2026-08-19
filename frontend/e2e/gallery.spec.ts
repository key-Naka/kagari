import { readFile } from 'node:fs/promises'
import { createServer, type Server } from 'node:http'
import { resolve } from 'node:path'
import { expect, test, type Locator, type Page } from '@playwright/test'

let apiServer: Server

test.beforeAll(async () => {
  const galleryItems = JSON.parse(await readFile(
    resolve(import.meta.dirname, '../../backend/cmd/api/gallery_items.json'),
    'utf8',
  ))
  apiServer = createServer((request, response) => {
    response.setHeader('Access-Control-Allow-Origin', 'http://127.0.0.1:3001')
    response.setHeader('Content-Type', 'application/json')
    if (request.url !== '/api/v1/gallery-items') {
      response.writeHead(404)
      response.end('{"error":"not found"}')
      return
    }
    response.writeHead(200)
    response.end(JSON.stringify(galleryItems))
  })
  await new Promise<void>((resolveListen, reject) => {
    apiServer.once('error', reject)
    apiServer.listen(8080, '127.0.0.1', resolveListen)
  })
})

test.afterAll(async () => {
  await new Promise<void>((resolveClose, reject) => apiServer.close(error => error ? reject(error) : resolveClose()))
})

interface Point {
  x: number
  y: number
}

async function canvasBox(canvas: Locator) {
  await canvas.scrollIntoViewIfNeeded()
  const box = await canvas.boundingBox()
  expect(box).not.toBeNull()
  return box!
}

async function albumPositions(page: Page): Promise<Record<string, Point>> {
  return page.getByTestId('album-item').evaluateAll((items) => Object.fromEntries(
    items.map((item) => {
      const bounds = item.getBoundingClientRect()
      return [item.getAttribute('data-album-id') ?? '', { x: bounds.x, y: bounds.y }]
    }),
  ))
}

async function albumTransforms(page: Page): Promise<Record<string, string>> {
  return page.getByTestId('album-item').evaluateAll(items => Object.fromEntries(
    items.map(item => [
      item.getAttribute('data-album-id') ?? '',
      getComputedStyle(item).transform,
    ]),
  ))
}

async function dragMouse(page: Page, dx: number, dy: number): Promise<void> {
  const canvas = page.getByTestId('gallery-canvas')
  const box = await canvasBox(canvas)
  const start = { x: box.x + box.width / 2, y: box.y + box.height / 2 }
  await page.mouse.move(start.x, start.y)
  await page.mouse.down()
  await page.mouse.move(start.x + dx, start.y + dy, { steps: 8 })
  await page.mouse.up()
}

async function swipeTouch(page: Page, start: Point, end: Point): Promise<void> {
  const session = await page.context().newCDPSession(page)
  try {
    await session.send('Input.dispatchTouchEvent', {
      type: 'touchStart',
      touchPoints: [{ x: start.x, y: start.y }],
    })
    for (let step = 1; step <= 6; step++) {
      const progress = step / 6
      await session.send('Input.dispatchTouchEvent', {
        type: 'touchMove',
        touchPoints: [{
          x: start.x + (end.x - start.x) * progress,
          y: start.y + (end.y - start.y) * progress,
        }],
      })
    }
    await session.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
  } finally {
    await session.detach()
  }
}

function expectWrappedOnAxis(
  before: Record<string, Point>,
  after: Record<string, Point>,
  axis: keyof Point,
  direction: 1 | -1,
): void {
  const deltas = Object.keys(before).map(id => after[id]![axis] - before[id]![axis])
  expect(deltas.some(delta => Math.sign(delta) === direction)).toBe(true)
  expect(deltas.some(delta => Math.sign(delta) === -direction)).toBe(true)
}

test.beforeEach(async ({ page }) => {
  await page.goto('/gallery')
  await expect(page.getByRole('heading', { name: '无界相册' })).toBeVisible()
  const canvas = page.getByTestId('gallery-canvas')
  await expect(canvas).toHaveAttribute('data-ready', 'true')
  const dimensions = await canvas.evaluate(element => ({ width: element.clientWidth, height: element.clientHeight }))
  expect(dimensions.width).toBeGreaterThan(0)
  expect(dimensions.height).toBeGreaterThan(0)
})

for (const direction of [
  { name: '向右', dx: 520, dy: 0, axis: 'x' as const, sign: 1 as const },
  { name: '向左', dx: -520, dy: 0, axis: 'x' as const, sign: -1 as const },
  { name: '向下', dx: 0, dy: 420, axis: 'y' as const, sign: 1 as const },
  { name: '向上', dx: 0, dy: -420, axis: 'y' as const, sign: -1 as const },
]) {
  test(`鼠标${direction.name}拖拽时 Album Item 从对侧连续回绕`, async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '桌面项目验证鼠标四方向拖拽。')
    const before = await albumPositions(page)
    await dragMouse(page, direction.dx, direction.dy)
    const after = await albumPositions(page)

    expect(Object.keys(after)).toEqual(Object.keys(before))
    expectWrappedOnAxis(before, after, direction.axis, direction.sign)
  })
}

test('反复跨越边界仍保持固定 Album Item 节点数，松手后没有惯性', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', '桌面项目验证节点恒定与无惯性。')
  const items = page.getByTestId('album-item')
  const initialCount = await items.count()
  expect(initialCount).toBeGreaterThan(6)

  const drags: readonly (readonly [number, number])[] = [[620, 0], [0, -480], [-730, 0], [0, 560], [690, 410]]
  for (const [dx, dy] of drags) {
    await dragMouse(page, dx, dy)
    await expect(items).toHaveCount(initialCount)
  }

  const released = await albumPositions(page)
  await page.waitForTimeout(250)
  expect(await albumPositions(page)).toEqual(released)
})

test('单指可二维拖拽，且只在画布手势激活期间阻止页面滚动', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'mobile', '移动项目验证单指与滚动仲裁。')
  const canvas = page.getByTestId('gallery-canvas')
  const box = await canvasBox(canvas)
  const initialCount = await page.getByTestId('album-item').count()
  const before = await albumPositions(page)
  const scrollBeforeCanvasGesture = await page.evaluate(() => window.scrollY)
  const start = { x: box.x + box.width / 2, y: box.y + box.height / 2 }
  await swipeTouch(page, start, { x: start.x + 96, y: start.y + 72 })

  const after = await albumPositions(page)
  expect(Object.keys(after)).toEqual(Object.keys(before))
  expect(Object.keys(before).some(id => after[id]!.x !== before[id]!.x && after[id]!.y !== before[id]!.y)).toBe(true)
  expect(await page.evaluate(() => window.scrollY)).toBeCloseTo(scrollBeforeCanvasGesture, 0)
  await expect(page.getByTestId('album-item')).toHaveCount(initialCount)
  await expect(canvas).not.toHaveClass(/infinite-gallery__canvas--dragging/)

  await page.evaluate(() => window.scrollTo(0, 0))
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0)
  const viewport = page.viewportSize()!
  await swipeTouch(
    page,
    { x: 4, y: viewport.height - 80 },
    { x: 4, y: Math.max(80, viewport.height - 380) },
  )
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBeGreaterThan(0)
})

test('滚轮只滚动页面，不会平移 Album Item', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop', '桌面项目验证禁止滚轮平移。')
  const canvas = page.getByTestId('gallery-canvas')
  const box = await canvasBox(canvas)
  const before = await albumTransforms(page)
  const scrollBefore = await page.evaluate(() => window.scrollY)
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await page.mouse.wheel(0, 240)
  await expect.poll(() => page.evaluate(() => window.scrollY)).toBeGreaterThan(scrollBefore)
  const scrollAfter = await page.evaluate(() => window.scrollY)
  expect(scrollAfter - scrollBefore).toBeGreaterThan(0)
  expect(await albumTransforms(page)).toEqual(before)
})
