import { expect, test, type Page } from '@playwright/test'

async function waitForInteractionSystem(page: Page): Promise<void> {
  await expect(page.getByTestId('archive-grid-canvas')).toBeVisible({ timeout: 15000 })
}

async function canvasPixelSummary(page: Page): Promise<{
  visiblePixels: number
  purplePixels: number
  centerAlpha: number
}> {
  return page.getByTestId('archive-grid-canvas').evaluate((element) => {
    const canvas = element as HTMLCanvasElement
    const context = canvas.getContext('2d')
    if (!context) {
      return { visiblePixels: 0, purplePixels: 0, centerAlpha: 0 }
    }

    const ratio = canvas.width / window.innerWidth
    const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data
    let visiblePixels = 0
    let purplePixels = 0
    for (let index = 0; index < pixels.length; index += 4) {
      const red = pixels[index] ?? 0
      const blue = pixels[index + 2] ?? 0
      const alpha = pixels[index + 3] ?? 0
      if (alpha > 0) {
        visiblePixels += 1
      }
      if (alpha > 8 && blue - red > 20) {
        purplePixels += 1
      }
    }

    const center = context.getImageData(
      Math.round(140 * ratio),
      Math.round(364 * ratio),
      1,
      1,
    ).data
    return {
      visiblePixels,
      purplePixels,
      centerAlpha: center[3] ?? 0,
    }
  })
}

test.describe('全站视觉交互系统', () => {
  test('桌面 Canvas 淡出后可重复激活，并在空闲时休眠', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面条件下验证 Canvas。')
    await page.goto('/')
    await waitForInteractionSystem(page)

    const canvas = page.getByTestId('archive-grid-canvas')
    await page.mouse.move(120, 360)
    await expect.poll(async () => Number(await canvas.getAttribute('data-activation-count'))).toBeGreaterThan(0)
    await expect.poll(async () => Number(await canvas.getAttribute('data-active-count'))).toBeGreaterThan(0)
    await expect(canvas).toHaveAttribute('data-render-state', 'running')
    await expect.poll(async () => (await canvasPixelSummary(page)).purplePixels).toBeGreaterThan(0)
    expect((await canvasPixelSummary(page)).centerAlpha).toBe(0)

    await expect.poll(async () => Number(await canvas.getAttribute('data-active-count'))).toBe(0)
    await expect(canvas).toHaveAttribute('data-render-state', 'idle')

    await page.mouse.move(320, 360)
    await expect.poll(async () => Number(await canvas.getAttribute('data-active-count'))).toBe(0)
    const secondFadedCount = Number(await canvas.getAttribute('data-activation-count'))
    await page.mouse.move(120, 360)
    await expect.poll(async () => Number(await canvas.getAttribute('data-activation-count'))).toBeGreaterThan(secondFadedCount)
    await expect(canvas).toHaveAttribute('data-render-state', 'running')
  })

  test('桌面 Canvas 在导航区域和不同公开路由上均响应指针', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面条件下验证 Canvas。')
    await page.goto('/')
    await waitForInteractionSystem(page)

    const canvas = page.getByTestId('archive-grid-canvas')
    const navigation = page.getByTestId('site-navigation')
    const navigationBox = await navigation.boundingBox()
    expect(navigationBox).not.toBeNull()
    const initialCount = Number(await canvas.getAttribute('data-activation-count'))
    await page.mouse.move(
      (navigationBox?.x ?? 0) + (navigationBox?.width ?? 0) / 2,
      (navigationBox?.y ?? 0) + (navigationBox?.height ?? 0) / 2,
    )
    await expect.poll(async () => Number(await canvas.getAttribute('data-activation-count'))).toBeGreaterThan(initialCount)

    await page.goto('/music')
    await waitForInteractionSystem(page)
    const musicCanvas = page.getByTestId('archive-grid-canvas')
    const musicCount = Number(await musicCanvas.getAttribute('data-activation-count'))
    await page.mouse.move(240, 420)
    await expect.poll(async () => Number(await musicCanvas.getAttribute('data-activation-count'))).toBeGreaterThan(musicCount)
  })

  test('reduced motion 仅保留静态 Canvas 网格', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面浏览器中模拟 reduced motion。')
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.goto('/')
    await waitForInteractionSystem(page)

    const canvas = page.getByTestId('archive-grid-canvas')
    await expect(canvas).toHaveAttribute('data-render-state', 'static')
    const staticPixels = await canvasPixelSummary(page)
    expect(staticPixels.visiblePixels).toBeGreaterThan(0)
    expect(staticPixels.purplePixels).toBe(0)
    await page.mouse.move(120, 360)
    await page.mouse.move(320, 360)
    await expect(canvas).toHaveAttribute('data-activation-count', '0')
    await expect(canvas).toHaveAttribute('data-active-count', '0')
    expect((await canvasPixelSummary(page)).purplePixels).toBe(0)
  })

  test('桌面导航滚动收紧且准星锁定真实目标', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面条件下验证导航与准星。')
    await page.goto('/')
    await waitForInteractionSystem(page)

    const navigation = page.getByTestId('site-navigation')
    await page.evaluate(() => window.scrollTo(0, 80))
    await expect(navigation).toHaveClass(/site-navigation--compact/)

    const musicLink = page.getByRole('link', { name: '音乐' }).first()
    await musicLink.hover()
    const cursor = page.getByTestId('target-cursor')
    await expect(cursor).toHaveAttribute('data-locked', 'true')
    await page.waitForTimeout(200)
    const targetBox = await musicLink.boundingBox()
    const cursorBox = await cursor.boundingBox()
    expect(cursorBox?.width).toBeCloseTo(targetBox?.width ?? 0, 0)
    expect(cursorBox?.height).toBeCloseTo(targetBox?.height ?? 0, 0)
  })

  test('桌面路由和浏览器后退都会经过统一遮罩转场', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面条件下验证路由遮罩。')
    await page.goto('/')
    await waitForInteractionSystem(page)
    const transition = page.getByTestId('page-transition')
    const musicLink = page.getByRole('link', { name: '音乐' }).first()

    await Promise.all([
      page.waitForURL('**/music'),
      musicLink.click(),
    ])
    await expect(transition).toHaveAttribute('data-sequence', '1')
    await expect(transition).toHaveAttribute('data-phase', 'idle')

    await page.goBack()
    await expect(page).toHaveURL(/\/$/)
    await expect(transition).toHaveAttribute('data-sequence', '2')
    await expect(transition).toHaveAttribute('data-phase', 'idle')

    await page.goForward()
    await expect(page).toHaveURL(/\/music$/)
    await expect(transition).toHaveAttribute('data-sequence', '3')
    await expect(transition).toHaveAttribute('data-phase', 'idle')
  })

  test('移动端不渲染准星，菜单锁定焦点并通过 Escape 恢复触发器', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'mobile', '仅在移动端验证菜单。')
    await page.goto('/')
    await waitForInteractionSystem(page)

    await expect(page.getByTestId('target-cursor')).toHaveCount(0)
    const noHorizontalOverflow = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
    expect(noHorizontalOverflow).toBe(true)
    const trigger = page.getByRole('button', { name: '打开全部导航' })
    await trigger.click()
    const menu = page.getByTestId('ritual-menu')
    await expect(menu).toHaveAttribute('data-open', 'true')
    await expect(page.getByRole('button', { name: '关闭导航' })).toBeFocused()

    await page.keyboard.press('Tab')
    await expect(page.getByRole('link', { name: '首页' }).last()).toBeFocused()
    await page.keyboard.press('Escape')
    await expect(menu).toHaveAttribute('data-open', 'false')
    await expect(menu).toHaveAttribute('inert', '')
    await expect(trigger).toBeFocused()
    await page.keyboard.press('Tab')
    await expect(page.locator('.ritual-menu__close')).not.toBeFocused()

    await trigger.click()
    const menuMusicLink = page.getByRole('link', { name: '音乐' }).last()
    await Promise.all([
      page.waitForURL('**/music'),
      menuMusicLink.click(),
    ])
    await expect(page.getByTestId('page-transition')).toHaveAttribute('data-sequence', '1')
    await expect(page.getByTestId('page-transition')).toHaveAttribute('data-phase', 'idle')
  })
})
