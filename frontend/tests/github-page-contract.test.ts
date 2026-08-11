import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

async function readFrontendFile(path: string): Promise<string> {
  return readFile(resolve(process.cwd(), path), 'utf8')
}

describe('GitHub 公开页面契约', () => {
  it('通过公开 API 展示贡献热力图、近期活动和仓库，并提供加载与降级状态', async () => {
    const source = await readFrontendFile('pages/github/index.vue')

    expect(source).toContain('<script setup lang="ts">')
    expect(source).toContain('useRuntimeConfig()')
    expect(source).toContain('useAsyncData')
    expect(source).toContain('$fetch<unknown>(`${apiBase}/api/v1/github`)')
    expect(source).toContain('parseGitHubActivityData')
    expect(source).toContain('role="status"')
    expect(source).toContain('role="alert"')
    expect(source).toContain('contributions')
    expect(source).toContain('activities')
    expect(source).toContain('repositories')
    expect(source).toContain('target="_blank"')
    expect(source).toContain('rel="noopener noreferrer"')
    expect(source).not.toContain("credentials: 'include'")
    expect(source).not.toContain('credentials: "include"')
  })
})
