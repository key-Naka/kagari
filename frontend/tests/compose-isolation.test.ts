import { describe, expect, it } from 'vitest'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

describe('Compose 服务隔离 seam', () => {
  it('将宿主机和 Docker 权限限制为各自唯一的私网代理服务', async () => {
    const compose = await readFile(resolve(import.meta.dirname, '../../docker-compose.yml'), 'utf8')
    const serviceBlock = (name: string) => {
      const match = compose.match(new RegExp(`^  ${name}:\\n([\\s\\S]*?)(?=^  [a-z][a-z-]*:|^networks:)`, 'm'))
      expect(match, `缺少 ${name} 服务定义`).not.toBeNull()
      return match?.[1] || ''
    }

    const frontend = serviceBlock('frontend')
    const backend = serviceBlock('backend')
    const hostMetrics = serviceBlock('host-metrics')
    const proxy = serviceBlock('docker-proxy')

    expect(frontend).toContain('"${FRONTEND_PORT:-127.0.0.1:3000}:3000"')
    expect(backend).toContain('"${BACKEND_PORT:-127.0.0.1:18080}:8080"')
    expect(backend).toContain('HOST_METRICS_URL: ${HOST_METRICS_URL:-http://host-metrics:8090}')
    expect(backend).toContain('STATUS_HTTP_URL: ${STATUS_HTTP_URL:-http://frontend:3000/}')
    expect(backend).toContain('host-metrics:\n        condition: service_started')
    expect(backend).not.toMatch(/^    volumes:/m)
    expect(backend).not.toMatch(/\/var\/run\/docker\.sock|\/host(?:\/|:)/)
    expect(backend).toContain('read_only: true')
    expect(backend).toContain('no-new-privileges:true')
    expect(backend).toContain('- ALL')

    expect(hostMetrics).toContain('- /proc:/host/proc:ro')
    expect(hostMetrics).toContain('- /proc/1/root:/host-root:ro')
    expect(hostMetrics).not.toMatch(/^    ports:/m)
    expect(hostMetrics).toContain('- private')
    expect(hostMetrics).toContain('read_only: true')
    expect(hostMetrics).toContain('no-new-privileges:true')
    expect(hostMetrics).toContain('- ALL')

    expect(proxy).toContain('tecnativa/docker-socket-proxy@sha256:1f5038b54f06c3e18422902cf00ba21803d1c97805aae032e5e6673d532d3459')
    expect(proxy).toContain('CONTAINERS: "1"')
    expect(proxy).toContain('POST: "0"')
    expect(proxy).toContain('- /var/run/docker.sock:/var/run/docker.sock:ro')
    expect(proxy).not.toMatch(/^    ports:/m)
    expect(proxy).toContain('- private')
    expect(proxy).toContain('no-new-privileges:true')
    expect(proxy).toContain('- ALL')

    expect(compose).toMatch(/^name: kagari$/m)
    expect(compose.match(/\/var\/run\/docker\.sock:\/var\/run\/docker\.sock:ro/g) || []).toHaveLength(1)
    expect(compose.match(/- \/proc:\/host\/proc:ro/g) || []).toHaveLength(1)
    expect(compose.match(/- \/proc\/1\/root:\/host-root:ro/g) || []).toHaveLength(1)
    expect(compose).toMatch(/^  private:\n    driver: bridge\n    internal: true/m)
  })
})
