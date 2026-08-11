import { access, readFile } from 'node:fs/promises'
import { constants } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const frontendRoot = resolve(import.meta.dirname, '..')

async function readFrontendFile(path: string): Promise<string> {
  return readFile(resolve(frontendRoot, path), 'utf8')
}

describe('公开 Service Status 路由契约', () => {
  it('提供 SSR 安全的 /status 页面、匿名 public API 请求和脱敏 DTO 展示', async () => {
    const source = await readFrontendFile('pages/status/index.vue')

    expect(source).toContain('<script setup lang="ts">')
    expect(source).toContain("path: '/status'")
    expect(source).toContain('useRuntimeConfig()')
    expect(source).toContain('runtimeConfig.public.apiBase')
    expect(source).toContain('fetch(`${apiBase}/api/v1/service-status`)')
    expect(source).toContain('parseServiceStatus')
    expect(source).toContain("'operational' | 'degraded' | 'unavailable'")
    expect(source).toContain('onMounted')
    expect(source).not.toContain('credentials: \'include\'')
    expect(source).not.toContain('credentials: "include"')
    expect(source).toContain('CPU')
    expect(source).toContain('内存')
    expect(source).toContain('磁盘')
    expect(source).toContain('网络')
    expect(source).toContain('运行时间')
    expect(source).toContain('Web')
    expect(source).toContain('API')
    expect(source).toContain('Database')
    expect(source).toContain('Cache')
    expect(source).toContain('HTTP')
    expect(source).toContain('MySQL')
    expect(source).toContain('Redis')
    expect(source).toContain('resources: AvailabilityState')
    expect(source).toContain('item.resources')
    expect(source).toContain('资源 {{ stateLabel(container.resources) }}')
    expect(source).toContain('采样时间')
    expect(source).toContain('role="status"')
    expect(source).toContain('role="alert"')
    expect(source).toContain('刷新状态')
    expect(source).not.toMatch(/\b(hostname|containerId|container_id|environment|command|port)\b/i)
  })

  it('首页包含 /status 公开导航入口', async () => {
    const source = await readFrontendFile('pages/index.vue')

    expect(source).toContain('to="/status"')
  })

  it('不再保留旧的 /service-status 页面文件', async () => {
    await expect(access(resolve(frontendRoot, 'pages/service-status/index.vue'), constants.F_OK)).rejects.toThrow()
  })
})
