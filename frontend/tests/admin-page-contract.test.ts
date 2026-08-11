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
})
