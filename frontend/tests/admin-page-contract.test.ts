import { describe, expect, it } from 'vitest'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

describe('管理控制台契约', () => {
  it('通过运行时 API 地址以 Cookie 会话管理站点配置', async () => {
    const source = await readFile(resolve(import.meta.dirname, '../pages/admin/index.vue'), 'utf8')

    expect(source).toContain('<script setup lang="ts">')
    expect(source).toContain('useRuntimeConfig()')
    expect(source).toContain("credentials: 'include'")
    expect(source).toContain("'/api/v1/admin/session'")
    expect(source).toContain("'/api/v1/admin/site-config'")
    expect(source).toContain("method: 'POST'")
    expect(source).toContain("method: 'PUT'")
    expect(source).toContain("method: 'DELETE'")
    expect(source).toContain("'Content-Type': 'application/json'")
    expect(source).toContain('JSON.parse(configurationText.value)')
  })
})
