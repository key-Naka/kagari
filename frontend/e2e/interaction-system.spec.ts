import { expect, test, type Page } from '@playwright/test'

async function waitForInteractionSystem(page: Page): Promise<void> {
  await expect(page.getByTestId('archive-grid-canvas')).toBeVisible({ timeout: 15000 })
}

test.describe('全站视觉交互系统', () => {
  test('桌面 Canvas 仅在进入新格子时激活，导航滚动收紧且准星锁定真实目标', async ({ page }, testInfo) => {
    test.skip(testInfo.project.name !== 'desktop', '仅在桌面条件下验证 Canvas 与准星。')
    await page.goto('/')
    await waitForInteractionSystem(page)

    const canvas = page.getByTestId('archive-grid-canvas')
    await page.mouse.move(120, 360)
    const firstCount = Number(await canvas.getAttribute('data-activation-count'))
    await expect.poll(async () => Number(await canvas.getAttribute('data-active-count'))).toBeGreaterThan(0)
    expect(Number(await canvas.getAttribute('data-active-count'))).toBeLessThanOrEqual(9)
    await page.mouse.move(121, 361)
    await expect(canvas).toHaveAttribute('data-activation-count', String(firstCount))
    await page.mouse.move(180, 360)
    await expect.poll(async () => Number(await canvas.getAttribute('data-activation-count'))).toBeGreaterThan(firstCount)
    await page.waitForTimeout(2900)
    await expect(canvas).toHaveAttribute('data-active-count', '0')

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
