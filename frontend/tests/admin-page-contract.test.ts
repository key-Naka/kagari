import { describe, expect, it } from 'vitest'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

async function readFrontendFile(path: string): Promise<string> {
  return readFile(resolve(process.cwd(), path), 'utf8')
}

describe('管理认证路由契约', () => {
  it('/admin/login 使用运行时 API 地址创建 Cookie 会话并跳转控制台', async () => {
    const source = await readFrontendFile('pages/admin/login/index.vue')

    expect(source).toContain('<script setup lang="ts">')
    expect(source).toContain("middleware: 'admin-auth'")
    expect(source).toContain('useRuntimeConfig()')
    expect(source).toContain('fetch(`${apiBase}/api/v1/admin/session`')
    expect(source).toContain("method: 'POST'")
    expect(source).toContain("'Content-Type': 'application/json'")
    expect(source).toContain("credentials: 'include'")
    expect(source).toContain("navigateTo('/admin')")
    expect(source).toContain('role="alert"')
  })

  it('命名中间件通过会话 API 重定向未认证用户和已认证登录用户', async () => {
    const source = await readFrontendFile('middleware/admin-auth.ts')

    expect(source).toContain('defineNuxtRouteMiddleware')
    expect(source).toContain('fetch(`${apiBase}/api/v1/admin/session`')
    expect(source).toContain("credentials: 'include'")
    expect(source).toContain("to.path === '/admin/login'")
    expect(source).toContain("return navigateTo('/admin')")
    expect(source).toContain("if (to.path !== '/admin/login')")
    expect(source).toContain("return navigateTo('/admin/login')")
  })

  it('控制台统一暴露所有内容入口并提供 Album Item 运营闭环', async () => {
    const [consolePage, galleryPage, mediaPage, mediaUpload, projectsPage] = await Promise.all([
      readFrontendFile('pages/admin/index.vue'),
      readFrontendFile('pages/admin/gallery-items.vue'),
      readFrontendFile('pages/admin/media.vue'),
      readFrontendFile('composables/useAdminMediaUpload.ts'),
      readFrontendFile('pages/admin/projects.vue'),
    ])

    for (const route of ['/admin/projects', '/admin/posts', '/admin/tracks', '/admin/gallery-items', '/admin/media', '/admin/visitor-messages']) {
      expect(consolePage).toContain(`to="${route}"`)
    }
    expect(galleryPage).toContain("definePageMeta({ middleware: 'admin-auth' })")
    expect(galleryPage).toContain("requestApi('/api/v1/admin/gallery-items')")
    expect(galleryPage).toContain('useAdminMediaUpload()')
    expect(mediaUpload).toContain("requestApi('/api/v1/admin/media/upload-credentials'")
    expect(mediaUpload).toContain("requestApi('/api/v1/admin/media'")
    expect(mediaUpload).toContain("upload.append('token', credentials.uploadToken)")
    expect(mediaPage).toContain("requestApi('/api/v1/admin/media')")
    expect(mediaPage).toContain('audio/mpeg,audio/ogg,audio/wav')
    expect(mediaPage).not.toContain('audio/mp4')
    expect(projectsPage).toContain('useAdminApi()')
    expect(projectsPage).toContain(':key="coverInputKey"')
    expect(galleryPage).toContain("method: activeItemId.value === null ? 'POST' : 'PUT'")
    expect(galleryPage).toContain("method: 'DELETE'")
    expect(galleryPage).toContain('window.confirm')
    expect(galleryPage).toContain(':key="fileInputKey"')
    expect(galleryPage).toContain('onMounted(loadItems)')
    expect(galleryPage).toContain('还没有 Album Item')
    expect(galleryPage).toContain('role="status"')
    expect(galleryPage).toContain('role="alert"')
  })

  it('结构化站点配置控制公开首页 SEO 与分享图', async () => {
    const [consolePage, homePage] = await Promise.all([
      readFrontendFile('pages/admin/index.vue'),
      readFrontendFile('pages/index.vue'),
    ])

    expect(consolePage).toContain('v-model.trim="configuration.siteTitle"')
    expect(consolePage).toContain('v-model.trim="configuration.seoSummary"')
    expect(consolePage).toContain('v-model.trim="configuration.shareImageUrl"')
    expect(consolePage).toContain(':key="shareImageInputKey"')
    expect(consolePage).toContain("requestApi('/api/v1/admin/site-config'")
    expect(homePage).toContain('/api/v1/site-config')
    expect(homePage).toContain('title: () => siteConfig.value.siteTitle')
    expect(homePage).toContain('image: () => siteConfig.value.shareImageUrl || undefined')
  })
})
