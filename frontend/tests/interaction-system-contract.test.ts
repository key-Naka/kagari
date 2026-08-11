import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

async function readFrontendFile(path: string): Promise<string> {
  return readFile(resolve(process.cwd(), path), 'utf8')
}

describe('全站视觉交互系统契约', () => {
  it('由独立的 Canvas、准星、导航和转场组件组成，并在 app 壳层装配', async () => {
    const app = await readFrontendFile('app.vue')
    const canvas = await readFrontendFile('components/ArchiveGridCanvas.client.vue')
    const cursor = await readFrontendFile('components/TargetCursor.client.vue')
    const navigation = await readFrontendFile('components/SiteNavigation.vue')
    const overlay = await readFrontendFile('components/PageTransitionOverlay.vue')

    expect(app).toContain('<ArchiveGridCanvas')
    expect(app).toContain('<TargetCursor')
    expect(app).toContain('<SiteNavigation')
    expect(app).toContain('<PageTransitionOverlay')
    expect(canvas).toContain('requestAnimationFrame')
    expect(canvas).toContain('visitedCells')
    expect(canvas).toContain('activationDuration = 2500')
    expect(cursor).toContain("matchMedia('(hover: hover) and (pointer: fine)')")
    expect(cursor).toContain("closest<HTMLElement>('.cursor-target')")
    expect(navigation).toContain('window.scrollY > 32')
    expect(navigation).toContain("event.key === 'Escape'")
    expect(navigation).toContain('document.body.style.overflow = \'hidden\'')
    expect(overlay).toContain('data-testid="page-transition"')
  })

  it('客户端路由插件在导航前遮挡、目标页面准备后退场', async () => {
    const plugin = await readFrontendFile('plugins/page-transition.client.ts')
    const transition = await readFrontendFile('composables/usePageTransition.ts')

    expect(plugin).toContain('router.beforeEach')
    expect(plugin).toContain('await transition.cover()')
    expect(plugin).toContain('router.afterEach')
    expect(plugin).toContain('transition.reveal()')
    expect(transition).toContain("type PageTransitionPhase = 'idle' | 'covering' | 'covered' | 'revealing'")
  })
})
