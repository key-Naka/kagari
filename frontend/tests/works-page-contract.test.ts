import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

async function readFrontendFile(path: string): Promise<string> {
  return readFile(resolve(process.cwd(), path), 'utf8')
}

describe('Portfolio Project 页面契约', () => {
  it('公开作品目录与详情页通过 SSR 公开 API 渲染已发布作品和安全外链', async () => {
    const listPage = await readFrontendFile('pages/works/index.vue')
    const detailPage = await readFrontendFile('pages/works/[slug].vue')

    expect(listPage).toContain('<script setup lang="ts">')
    expect(listPage).toContain('useAsyncData')
    expect(listPage).toContain('$fetch<unknown>(`${apiBase}/api/v1/projects`)')
    expect(listPage).toContain('parseProjectList')
    expect(listPage).toContain('/works/${project.slug}')
    expect(listPage).toContain('role="alert"')

    expect(detailPage).toContain('useRoute()')
    expect(detailPage).toContain('useAsyncData')
    expect(detailPage).toContain('/api/v1/projects/${slug}')
    expect(detailPage).toContain('parseProject')
    expect(detailPage).toContain('rel="noopener noreferrer"')
  })

  it('管理员作品页使用受保护会话完成创建、编辑、发布和永久删除', async () => {
    const adminPage = await readFrontendFile('pages/admin/projects.vue')
    const dashboard = await readFrontendFile('pages/admin/index.vue')

    expect(adminPage).toContain("definePageMeta({ middleware: 'admin-auth' })")
    expect(adminPage).toContain("requestApi('/api/v1/admin/projects')")
    expect(adminPage).toContain("method: activeProjectId.value === null ? 'POST' : 'PUT'")
    expect(adminPage).toContain("method: 'DELETE'")
    expect(adminPage).toContain('window.confirm')
    expect(adminPage).toContain('credentials: \'include\'')
    expect(dashboard).toContain('to="/admin/projects"')
  })
})
