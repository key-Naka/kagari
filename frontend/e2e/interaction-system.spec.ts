import { expect, test, type Page } from '@playwright/test'

const publicRoutes = [
  { label: '首页', path: '/' },
  { label: '作品', path: '/works' },
  { label: '博客', path: '/blog' },
  { label: '音乐', path: '/music' },
  { label: '相册', path: '/gallery' },
  { label: 'GitHub', path: '/github' },
  { label: '服务状态', path: '/status' },
  { label: '访客留言', path: '/visitor-messages' },
]

async function waitForInteractionSystem(page: Page): Promise<void> {
  await expect(page.getByTestId('archive-grid-canvas')).toBeVisible({ timeout: 15000 })
}

async function expectNavigationClearOfContent(page: Page): Promise<void> {
  const geometry = await page.evaluate(() => {
    const navigation = document.querySelector<HTMLElement>('[data-testid="site-navigation"]')
    const main = document.querySelector<HTMLElement>('main')
    if (!navigation || !main) {
      return null
    }
    const navigationBounds = navigation.getBoundingClientRect()
    const mainBounds = main.getBoundingClientRect()
    return {
      navigationBottom: navigationBounds.bottom,
      mainTop: mainBounds.top,
    }
  })
  expect(geometry).not.toBeNull()
  expect(geometry?.mainTop ?? 0).toBeGreaterThanOrEqual(geometry?.navigationBottom ?? Number.POSITIVE_INFINITY)
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

async function rotationDegrees(page: Page): Promise<number> {
  return page.locator('.target-cursor__ring').evaluate((element) => {
    const matrix = new DOMMatrixReadOnly(getComputedStyle(element).transform)
    const degrees = Math.atan2(matrix.b, matrix.a) * 180 / Math.PI
    return degrees < 0 ? degrees + 360 : degrees
  })
}

function clockwiseDelta(from: number, to: number): number {
  return (to - from + 360) % 360
}

function angularDistance(from: number, to: number): number {
  const delta = Math.abs(from - to) % 360
  return Math.min(delta, 360 - delta)
}

async function cursorPoint(page: Page): Promise<{ x: number; y: number }> {
  return page.getByTestId('target-cursor').evaluate((element) => {
    const bounds = element.getBoundingClientRect()
    return {
      x: bounds.left,
      y: bounds.top,
    }
  })
}

async function centerDotScale(page: Page): Promise<number> {
  return page.locator('.target-cursor__dot').evaluate((element) => {
    return new DOMMatrixReadOnly(getComputedStyle(element).transform).a
  })
}

async function cursorScale(page: Page): Promise<number> {
  return page.getByTestId('target-cursor').evaluate((element) => {
    return new DOMMatrixReadOnly(getComputedStyle(element).transform).a
  })
}

async function cornerAnchorPoints(page: Page): Promise<Record<string, { x: number; y: number }>> {
  return page.locator('[data-cursor-corner]').evaluateAll((elements) => Object.fromEntries(
    elements.map((element) => {
      const bounds = element.getBoundingClientRect()
      const name = (element as HTMLElement).dataset.cursorCorner ?? ''
      return [
        name,
        {
          x: bounds.left + bounds.width / 2,
          y: bounds.top + bounds.height / 2,
        },
      ]
    }),
  ))
}

async function cornerElbowOffsets(page: Page): Promise<Record<string, { x: number; y: number }>> {
  return page.locator('[data-cursor-corner]').evaluateAll((elements) => Object.fromEntries(
    elements.map((element) => {
      const name = (element as HTMLElement).dataset.cursorCorner ?? ''
      const mark = element.querySelector<HTMLElement>('.target-cursor__corner-mark')
      if (!mark) {
        return [name, { x: Number.NaN, y: Number.NaN }]
      }
      const matrix = new DOMMatrixReadOnly(getComputedStyle(mark).transform)
      const localX = name.includes('right') ? mark.offsetWidth : 0
      const localY = name.includes('bottom') ? mark.offsetHeight : 0
      const point = new DOMPoint(localX, localY).matrixTransform(matrix)
      return [name, { x: point.x, y: point.y }]
    }),
  ))
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
    await expect(page.getByTestId('target-cursor')).toHaveCount(0)
    await expect(page.locator('html')).not.toHaveClass(/kagari-target-cursor/)
    await expect(canvas).toHaveAttribute('data-render-state', 'static')
    const staticPixels = await canvasPixelSummary(page)
    expect(staticPixels.visiblePixels).toBeGreaterThan(0)
    expect(staticPixels.purplePixels).toBe(0)
    await page.mouse.move(120, 360)
    await page.mouse.move(320, 360)
    await expect(canvas).toHaveAttribute('data-activation-count', '0')
    await expect(canvas).toHaveAttribute('data-active-count', '0')
    expect((await canvasPixelSummary(page)).purplePixels).toBe(0)

    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await expect(page.getByTestId('target-cursor')).toBeVisible()
    await expect(page.locator('html')).toHaveClass(/kagari-target-cursor/)
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await expect(page.getByTestId('target-cursor')).toHaveCount(0)
    await expect(page.locator('html')).not.toHaveClass(/kagari-target-cursor/)
    const restoredCursorStyles = await page.evaluate(() => ({
      body: getComputedStyle(document.body).cursor,
      link: getComputedStyle(document.querySelector('.site-navigation__route')!).cursor,
    }))
    expect(restoredCursorStyles.body).not.toBe('none')
    expect(restoredCursorStyles.link).not.toBe('none')
  })

  test('reduced motion 下菜单与页面转场近乎即时', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面浏览器中模拟 reduced motion。')
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.setViewportSize({ width: 1119, height: 720 })
    await page.goto('/')
    await waitForInteractionSystem(page)

    const trigger = page.getByRole('button', { name: '打开全部导航' })
    const menu = page.getByTestId('ritual-menu')
    const transition = page.getByTestId('page-transition')
    const durations = await page.evaluate(() => ({
      menu: getComputedStyle(document.querySelector<HTMLElement>('[data-testid="ritual-menu"]')!)
        .transitionDuration,
      transition: getComputedStyle(document.querySelector<HTMLElement>('[data-testid="page-transition"]')!)
        .transitionDuration,
    }))
    expect(durations.menu.split(',').every(value => Number.parseFloat(value) <= 0.01)).toBe(true)
    expect(durations.transition.split(',').every(value => Number.parseFloat(value) <= 0.01)).toBe(true)

    await trigger.click()
    await expect(menu).toHaveAttribute('data-open', 'true')
    const startedAt = Date.now()
    await menu.locator('.ritual-menu__route[href="/music"]').click()
    await expect(page).toHaveURL(/\/music$/)
    await expect(transition).toHaveAttribute('data-phase', 'idle')
    expect(Date.now() - startedAt).toBeLessThan(750)
  })

  test('桌面导航直接展示八个公开入口并支持新增路由转场', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面条件下验证完整导航。')
    await page.goto('/')
    await waitForInteractionSystem(page)

    const routes = page.locator('.site-navigation__routes .site-navigation__route')
    await expect(routes).toHaveCount(publicRoutes.length)
    await expect(page.locator('.site-navigation__more')).toHaveCount(0)
    await expect(page.getByRole('button', { name: '打开更多导航' })).toHaveCount(0)

    for (const route of publicRoutes) {
      const link = page.locator(`.site-navigation__route[href="${route.path}"]`)
      await expect(link).toBeVisible()
      await expect(link).toHaveText(route.label)
    }

    const transition = page.getByTestId('page-transition')
    await page.locator('.site-navigation__route[href="/status"]').click()
    await expect(page).toHaveURL(/\/status$/)
    await expect(transition).toHaveAttribute('data-sequence', '1')
    await expect(transition).toHaveAttribute('data-phase', 'idle')

    await page.goBack()
    await expect(page).toHaveURL(/\/$/)
    await expect(transition).toHaveAttribute('data-phase', 'idle')
    const sequenceBeforeVisitorRoute = Number(await transition.getAttribute('data-sequence'))
    await page.locator('.site-navigation__route[href="/visitor-messages"]').click()
    await expect(page).toHaveURL(/\/visitor-messages$/)
    await expect.poll(async () => Number(await transition.getAttribute('data-sequence')))
      .toBeGreaterThan(sequenceBeforeVisitorRoute)
    await expect(transition).toHaveAttribute('data-phase', 'idle')
  })

  test('导航悬停、当前态和 reduced motion 下均保持可读', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面条件下验证导航视觉状态。')
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.goto('/')
    await waitForInteractionSystem(page)

    const activeRoute = page.locator('.site-navigation__route[href="/"]')
    await expect(activeRoute).toHaveClass(/site-navigation__route--active/)
    await expect(activeRoute).toHaveAttribute('aria-current', 'page')
    await expect(activeRoute.locator('.site-navigation__active-sigil')).toBeVisible()
    const activeBeforeHover = await activeRoute.evaluate(element => ({
      color: getComputedStyle(element).color,
      fill: getComputedStyle(element.querySelector<HTMLElement>('[data-liquid-fill]')!).backgroundColor,
    }))
    await activeRoute.hover()
    expect(await activeRoute.evaluate(element => ({
      color: getComputedStyle(element).color,
      fill: getComputedStyle(element.querySelector<HTMLElement>('[data-liquid-fill]')!).backgroundColor,
    }))).toEqual(activeBeforeHover)

    for (const route of publicRoutes.slice(1)) {
      const link = page.locator(`.site-navigation__route[href="${route.path}"]`)
      await link.hover()
      await expect.poll(async () => link.evaluate((element) => {
        const fill = element.querySelector<HTMLElement>('[data-liquid-fill]')
        return fill
          ? new DOMMatrixReadOnly(getComputedStyle(fill).transform).m42
          : Number.NaN
      })).toBeCloseTo(0, 5)
      const styles = await link.evaluate((element) => {
        const label = element.querySelector<HTMLElement>('.site-navigation__label')
        const fill = element.querySelector<HTMLElement>('[data-liquid-fill]')
        return {
          color: label ? getComputedStyle(label).color : '',
          opacity: label ? Number(getComputedStyle(label).opacity) : 0,
          fillColor: fill ? getComputedStyle(fill).backgroundColor : '',
        }
      })
      expect(styles.color).not.toBe('rgba(0, 0, 0, 0)')
      expect(styles.opacity).toBeGreaterThan(0)
      expect(styles.fillColor).not.toBe('rgba(0, 0, 0, 0)')
    }
  })

  test('导航在 1120px 以下和宽屏触屏环境切换为 Ritual Mobile Menu', async ({ page, browser }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '由桌面项目覆盖精确断点与宽屏触屏环境。')
    await page.setViewportSize({ width: 1121, height: 720 })
    await page.goto('/')
    await waitForInteractionSystem(page)
    await expect(page.locator('.site-navigation__routes')).toBeVisible()
    await expect(page.getByRole('button', { name: '打开全部导航' })).toBeHidden()
    await expectNavigationClearOfContent(page)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)

    await page.setViewportSize({ width: 1119, height: 720 })
    await expect(page.locator('.site-navigation__routes')).toBeHidden()
    await expect(page.getByRole('button', { name: '打开全部导航' })).toBeVisible()
    await expectNavigationClearOfContent(page)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)

    const touchContext = await browser.newContext({
      viewport: { width: 1366, height: 768 },
      hasTouch: true,
    })
    const touchPage = await touchContext.newPage()
    try {
      await touchPage.goto('/')
      await waitForInteractionSystem(touchPage)
      await expect(touchPage.locator('.site-navigation__routes')).toBeHidden()
      await expect(touchPage.getByRole('button', { name: '打开全部导航' })).toBeVisible()
      await expectNavigationClearOfContent(touchPage)
      expect(await touchPage.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    } finally {
      await touchContext.close()
    }
  })

  test('桌面 Target Cursor 隐藏系统指针，归正旋转并独立锁定四角', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面条件下验证导航与准星。')
    await page.goto('/')
    await waitForInteractionSystem(page)

    const cursor = page.getByTestId('target-cursor')
    await expect(cursor).toBeVisible()
    await expect(page.locator('html')).toHaveClass(/kagari-target-cursor/)
    const cursorStyles = await page.evaluate(() => ({
      body: getComputedStyle(document.body).cursor,
      link: getComputedStyle(document.querySelector('.site-navigation__route')!).cursor,
      label: getComputedStyle(document.querySelector('.site-navigation__label')!).cursor,
    }))
    expect(cursorStyles).toEqual({ body: 'none', link: 'none', label: 'none' })

    await page.mouse.move(160, 420)
    await expect.poll(async () => (await cursorPoint(page)).x).toBeCloseTo(160, 0)
    const followStartedAt = Date.now()
    await page.mouse.move(320, 420)
    await expect.poll(async () => (await cursorPoint(page)).x, {
      intervals: [20],
      timeout: 300,
    }).toBeCloseTo(320, 0)
    expect(Date.now() - followStartedAt).toBeLessThan(300)

    const freeRotation = await rotationDegrees(page)
    await page.waitForTimeout(260)
    const laterFreeRotation = await rotationDegrees(page)
    const rotationDelta = clockwiseDelta(freeRotation, laterFreeRotation)
    expect(rotationDelta).toBeGreaterThan(35)
    expect(rotationDelta).toBeLessThan(60)

    const musicLink = page.getByRole('link', { name: '音乐' }).first()
    for (const entryPhase of [30, 120, 240]) {
      await page.mouse.move(320, 520)
      await expect(cursor).toHaveAttribute('data-locked', 'false')
      await expect.poll(
        async () => angularDistance(await rotationDegrees(page), entryPhase),
        { intervals: [16], timeout: 2500 },
      ).toBeLessThan(8)

      await musicLink.hover()
      await expect(cursor).toHaveAttribute('data-locked', 'true')
      await expect.poll(
        async () => angularDistance(await rotationDegrees(page), 0),
        { intervals: [16], timeout: 300 },
      ).toBeLessThan(1)
      await page.waitForTimeout(260)
      expect(angularDistance(await rotationDegrees(page), 0)).toBeLessThan(1)

      const markBounds = await page.locator('.target-cursor__corner-mark').evaluateAll(
        marks => marks.map((mark) => {
          const bounds = mark.getBoundingClientRect()
          return { width: bounds.width, height: bounds.height }
        }),
      )
      for (const bounds of markBounds) {
        expect(bounds.width).toBeCloseTo(12, 1)
        expect(bounds.height).toBeCloseTo(12, 1)
      }
    }

    const lockedCursorStyles = await cursor.evaluate((element) => ({
      color: getComputedStyle(element).color,
      borderWidths: Array.from(
        element.querySelectorAll<HTMLElement>('.target-cursor__corner-mark'),
        mark => [
          getComputedStyle(mark).borderTopWidth,
          getComputedStyle(mark).borderRightWidth,
          getComputedStyle(mark).borderBottomWidth,
          getComputedStyle(mark).borderLeftWidth,
        ],
      ),
    }))
    expect(lockedCursorStyles.color).toBe('rgb(180, 151, 207)')
    for (const widths of lockedCursorStyles.borderWidths) {
      expect(widths.filter(width => width === '3px')).toHaveLength(2)
      expect(widths.filter(width => width === '0px')).toHaveLength(2)
    }

    const targetBox = await musicLink.boundingBox()
    expect(targetBox).not.toBeNull()
    const lockedCorners = await cornerAnchorPoints(page)
    const expectedCorners = {
      'top-left': { x: (targetBox?.x ?? 0) - 3, y: (targetBox?.y ?? 0) - 3 },
      'top-right': {
        x: (targetBox?.x ?? 0) + (targetBox?.width ?? 0) + 3,
        y: (targetBox?.y ?? 0) - 3,
      },
      'bottom-right': {
        x: (targetBox?.x ?? 0) + (targetBox?.width ?? 0) + 3,
        y: (targetBox?.y ?? 0) + (targetBox?.height ?? 0) + 3,
      },
      'bottom-left': {
        x: (targetBox?.x ?? 0) - 3,
        y: (targetBox?.y ?? 0) + (targetBox?.height ?? 0) + 3,
      },
    }
    for (const [name, expected] of Object.entries(expectedCorners)) {
      expect(lockedCorners[name]?.x).toBeCloseTo(expected.x, 0)
      expect(lockedCorners[name]?.y).toBeCloseTo(expected.y, 0)
    }
    const elbowOffsets = await cornerElbowOffsets(page)
    for (const offset of Object.values(elbowOffsets)) {
      expect(offset.x).toBeCloseTo(0, 5)
      expect(offset.y).toBeCloseTo(0, 5)
    }
    const cornerSizes = await page.locator('.target-cursor__corner-mark').evaluateAll(
      marks => marks.map(mark => ({
        width: (mark as HTMLElement).offsetWidth,
        height: (mark as HTMLElement).offsetHeight,
      })),
    )
    expect(cornerSizes).toEqual(Array.from({ length: 4 }, () => ({ width: 12, height: 12 })))

    const initialLeft = lockedCorners['top-left']?.x ?? 0
    await musicLink.evaluate((element) => {
      const width = element.getBoundingClientRect().width
      element.style.flex = '0 0 auto'
      element.style.width = `${width + 40}px`
    })
    const resizedTarget = await musicLink.boundingBox()
    expect(resizedTarget?.width).toBeGreaterThan((targetBox?.width ?? 0) + 30)
    await expect.poll(async () => (await cornerAnchorPoints(page))['top-right']?.x ?? 0)
      .toBeCloseTo((resizedTarget?.x ?? 0) + (resizedTarget?.width ?? 0) + 3, 0)
    const resizedCorners = await cornerAnchorPoints(page)
    expect(resizedCorners['top-left']?.x).not.toBeCloseTo(initialLeft, 0)
    expect(resizedCorners['top-left']?.x).toBeCloseTo((resizedTarget?.x ?? 0) - 3, 0)
    expect(resizedCorners['top-right']?.x).toBeCloseTo(
      (resizedTarget?.x ?? 0) + (resizedTarget?.width ?? 0) + 3,
      0,
    )

    const beforeViewportResizeLeft = resizedCorners['top-left']?.x ?? 0
    await page.setViewportSize({ width: 1400, height: 720 })
    await expect.poll(async () => (await cornerAnchorPoints(page))['top-left']?.x ?? 0)
      .not.toBeCloseTo(beforeViewportResizeLeft, 0)
    const viewportResizedTarget = await musicLink.boundingBox()
    const viewportResizedCorners = await cornerAnchorPoints(page)
    expect(viewportResizedCorners['top-left']?.x).toBeCloseTo((viewportResizedTarget?.x ?? 0) - 3, 0)

    const beforeScrollTop = lockedCorners['top-left']?.y ?? 0
    await page.evaluate(() => window.scrollTo(0, 80))
    await expect(page.getByTestId('site-navigation')).toHaveClass(/site-navigation--compact/)
    await expect.poll(async () => (await cornerAnchorPoints(page))['top-left']?.y ?? 0)
      .not.toBeCloseTo(beforeScrollTop, 0)
    await expect.poll(async () => page.evaluate(() => {
      const target = document.querySelector<HTMLElement>('.site-navigation__route[href="/music"]')
      const corner = document.querySelector<HTMLElement>('[data-cursor-corner="top-left"]')
      if (!target || !corner) {
        return Number.POSITIVE_INFINITY
      }
      const targetBounds = target.getBoundingClientRect()
      const cornerBounds = corner.getBoundingClientRect()
      const cornerY = cornerBounds.top + cornerBounds.height / 2
      return Math.abs(cornerY - (targetBounds.top - 3))
    })).toBeLessThan(0.5)

    await page.mouse.down()
    await expect.poll(async () => centerDotScale(page)).toBeLessThan(0.8)
    await expect.poll(async () => cursorScale(page)).toBeLessThan(0.95)
    await page.mouse.up()
    await expect.poll(async () => centerDotScale(page)).toBeCloseTo(1, 5)
    await expect.poll(async () => cursorScale(page)).toBeCloseTo(1, 5)

    await page.mouse.move(320, 520)
    await expect(cursor).toHaveAttribute('data-locked', 'false')
    const resumedRotation = await rotationDegrees(page)
    await page.waitForTimeout(260)
    expect(Math.abs(await rotationDegrees(page) - resumedRotation)).toBeGreaterThan(20)

    await page.goto('/admin/login')
    await expect(page.getByTestId('target-cursor')).toHaveCount(0)
    await expect(page.locator('html')).not.toHaveClass(/kagari-target-cursor/)
    expect(await page.evaluate(() => getComputedStyle(document.body).cursor)).not.toBe('none')
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
    await expect(page.locator('html')).not.toHaveClass(/kagari-target-cursor/)
    const noHorizontalOverflow = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
    expect(noHorizontalOverflow).toBe(true)
    const trigger = page.getByRole('button', { name: '打开全部导航' })
    await trigger.click()
    const menu = page.getByTestId('ritual-menu')
    await expect(menu).toHaveAttribute('data-open', 'true')
    await expect.poll(async () => page.evaluate(() => document.body.style.overflow)).toBe('hidden')
    const menuBounds = await menu.boundingBox()
    expect(menuBounds).not.toBeNull()
    expect(menuBounds?.x).toBeCloseTo(0, 0)
    expect(menuBounds?.y).toBeCloseTo(0, 0)
    expect(menuBounds?.width).toBeCloseTo(await page.evaluate(() => window.innerWidth), 0)
    expect(menuBounds?.height).toBeCloseTo(await page.evaluate(() => window.innerHeight), 0)
    await expect(page.getByRole('button', { name: '关闭导航' })).toBeFocused()

    await page.keyboard.press('Shift+Tab')
    await expect(menu.getByRole('link', { name: '访客留言' })).toBeFocused()
    await page.keyboard.press('Tab')
    await expect(page.getByRole('button', { name: '关闭导航' })).toBeFocused()
    await page.keyboard.press('Tab')
    await expect(menu.getByRole('link', { name: '首页' })).toBeFocused()
    await page.keyboard.press('Escape')
    await expect(menu).toHaveAttribute('data-open', 'false')
    await expect(menu).toHaveAttribute('inert', '')
    await expect.poll(async () => page.evaluate(() => document.body.style.overflow)).toBe('')
    await expect(trigger).toBeFocused()
    await page.keyboard.press('Tab')
    await expect(page.locator('.ritual-menu__close')).not.toBeFocused()

    await trigger.click()
    await expect(menu).toHaveAttribute('data-open', 'true')
    await menu.evaluate(element => (element as HTMLElement).click())
    await expect(menu).toHaveAttribute('data-open', 'false')
    await expect(trigger).toBeFocused()

    await trigger.click()
    const menuMusicLink = menu.getByRole('link', { name: '音乐' })
    await Promise.all([
      page.waitForURL('**/music'),
      menuMusicLink.click(),
    ])
    await expect(page.getByTestId('page-transition')).toHaveAttribute('data-sequence', '1')
    await expect(page.getByTestId('page-transition')).toHaveAttribute('data-phase', 'idle')
  })
})
