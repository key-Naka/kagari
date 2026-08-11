import { describe, expect, it } from 'vitest'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

describe('Compose 服务隔离 seam', () => {
  it('仅将 Web 与 API 绑定到回环地址，且数据服务限制在内部网络', async () => {
    const compose = await readFile(resolve(import.meta.dirname, '../../docker-compose.yml'), 'utf8')

    const serviceBlock = (name: string) => {
      const match = compose.match(new RegExp(`^  ${name}:\\n([\\s\\S]*?)(?=^  [a-z][a-z-]*:|^networks:)`, 'm'))
      expect(match, `缺少 ${name} 服务定义`).not.toBeNull()
      return match?.[1] || ''
    }

    expect(serviceBlock('frontend')).toContain('"${FRONTEND_PORT:-127.0.0.1:3000}:3000"')
    expect(serviceBlock('backend')).toContain('"${BACKEND_PORT:-127.0.0.1:18080}:8080"')
    expect(serviceBlock('frontend')).toContain('NUXT_PUBLIC_API_BASE: ${NUXT_PUBLIC_API_BASE:?NUXT_PUBLIC_API_BASE is required}')
    expect(serviceBlock('mysql')).not.toMatch(/^    ports:/m)
    expect(serviceBlock('redis')).not.toMatch(/^    ports:/m)
    expect(serviceBlock('backend')).toContain('- private')
    expect(serviceBlock('mysql')).toContain('- private')
    expect(serviceBlock('redis')).toContain('- private')
    expect(serviceBlock('redis')).toContain('REDIS_PASSWORD: ${REDIS_PASSWORD:?REDIS_PASSWORD is required}')
    expect(compose).toMatch(/^  private:\n    driver: bridge\n    internal: true/m)
  })
})
