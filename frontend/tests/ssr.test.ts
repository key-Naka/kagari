import { describe, expect, it } from 'vitest'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

describe('首页 SSR seam', () => {
  it('通过公开 API 输出作品导向摘要与全部独立模块入口', async () => {
    const [page, archive, config] = await Promise.all([
      readFile(resolve(import.meta.dirname, '../pages/index.vue'), 'utf8'),
      readFile(resolve(import.meta.dirname, '../utils/home-archive.ts'), 'utf8'),
      readFile(resolve(import.meta.dirname, '../nuxt.config.ts'), 'utf8'),
    ])

    expect(page).toContain('await useAsyncData')
    expect(page).toContain('全栈工程师 / 独立创作者')
    for (const route of ['/works', '/blog', '/music', '/github', '/gallery', '/status', '/visitor-messages']) {
      expect(page).toContain(`route: '${route}'`)
    }
    expect(archive).toContain('/api/v1/home')
    expect(page).toContain('...data.value.gallery')
    expect(page).not.toContain("headline: '12 个 Album Item'")
    expect(config).not.toContain("'/': { prerender: true }")
  })
})
