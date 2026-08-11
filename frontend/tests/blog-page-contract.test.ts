import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

async function readFrontendFile(path: string): Promise<string> {
  return readFile(resolve(process.cwd(), path), 'utf8')
}

describe('Blog Post 页面契约', () => {
  it('公开博客目录通过 SSR API 输出已发布文章、标签和年月归档筛选', async () => {
    const listPage = await readFrontendFile('pages/blog/index.vue')

    expect(listPage).toContain('<script setup lang="ts">')
    expect(listPage).toContain('useAsyncData')
    expect(listPage).toContain('/api/v1/posts${suffix}')
    expect(listPage).toContain('/api/v1/posts/tags')
    expect(listPage).toContain('/api/v1/posts/archives')
    expect(listPage).toContain('setFilters')
    expect(listPage).toContain('role="alert"')
  })

  it('文章详情只渲染服务端清理后的公开 HTML，后台使用认证接口管理完整 Markdown', async () => {
    const detailPage = await readFrontendFile('pages/blog/[slug].vue')
    const adminPage = await readFrontendFile('pages/admin/posts.vue')
    const dashboard = await readFrontendFile('pages/admin/index.vue')

    expect(detailPage).toContain('useRoute()')
    expect(detailPage).toContain('/api/v1/posts/${slug}')
    expect(detailPage).toContain("statusCode: 404")
    expect(detailPage).toContain('v-html="data.content"')
    expect(adminPage).toContain("definePageMeta({ middleware: 'admin-auth' })")
    expect(adminPage).toContain("requestApi('/api/v1/admin/posts')")
    expect(adminPage).toContain("method: activePostId.value === null ? 'POST' : 'PUT'")
    expect(adminPage).toContain("method: 'DELETE'")
    expect(adminPage).toContain("credentials: 'include'")
    expect(dashboard).toContain('to="/admin/posts"')
  })
})
