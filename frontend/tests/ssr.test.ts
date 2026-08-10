import { describe, expect, it } from 'vitest'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

describe('首页 SSR seam', () => {
  it('包含可由 SSR 输出的首页标题和导航入口', async () => {
    const source = await readFile(resolve(import.meta.dirname, '../pages/index.vue'), 'utf8')
    expect(source).toContain('创作、文字与声音的个人档案。')
    expect(source).toContain('to="/works"')
    expect(source).toContain('to="/admin"')
  })
})
