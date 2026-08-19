import { describe, expect, it } from 'vitest'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const repositoryRoot = resolve(import.meta.dirname, '../..')
const readRepositoryFile = async (path: string) => (
  await readFile(resolve(repositoryRoot, path), 'utf8')
).replaceAll('\r\n', '\n')

describe('生产发布契约', () => {
  it('为 1Panel 提供双域名 TLS 反向代理且只代理回环端口', async () => {
    const nginx = await readRepositoryFile('deploy/1panel/nginx/kagari.conf')
    const guide = await readRepositoryFile('deploy/1panel/README.md')

    expect(nginx).toMatch(/listen 443 ssl http2;/)
    expect(nginx).toMatch(/server_name ykagari\.top;/)
    expect(nginx).toMatch(/proxy_pass http:\/\/127\.0\.0\.1:3000;/)
    expect(nginx).toMatch(/server_name kagari-api\.ykagari\.top;/)
    expect(nginx).toMatch(/proxy_pass http:\/\/127\.0\.0\.1:18080;/)
    expect(nginx).toContain('ssl_certificate /etc/letsencrypt/live/ykagari.top/fullchain.pem;')
    expect(nginx).toContain('root /var/www/kagari-acme;')
    expect(nginx).toContain('proxy_set_header X-Forwarded-Proto $scheme;')
    expect(nginx).toContain('client_max_body_size 32m;')
    expect(guide).toContain('1Panel')
    expect(guide).toContain('docker compose')
    expect(guide).toContain('完整的 `nginx/kagari.conf`')
    expect(guide).toContain('`http` 上下文')
    expect(guide).toContain('https://ykagari.top/health')
    expect(guide).toContain('https://kagari-api.ykagari.top/health')
  })

  it('健康检查覆盖应用与依赖，数据服务和宿主代理不发布端口', async () => {
    const compose = await readRepositoryFile('docker-compose.yml')
    const serviceBlock = (name: string) => {
      const match = compose.match(new RegExp(`^  ${name}:\\n([\\s\\S]*?)(?=^  [a-z][a-z-]*:|^networks:)`, 'm'))
      expect(match, `缺少 ${name} 服务定义`).not.toBeNull()
      return match?.[1] || ''
    }

    for (const name of ['frontend', 'backend', 'host-metrics', 'docker-proxy', 'mysql', 'redis']) {
      expect(serviceBlock(name)).toContain('healthcheck:')
    }
    expect(serviceBlock('host-metrics')).toContain('[ "CMD", "wget", "--quiet", "--output-document=/dev/null", "http://127.0.0.1:8090/status" ]')
    expect(serviceBlock('host-metrics')).not.toContain('--spider')
    for (const name of ['host-metrics', 'docker-proxy', 'mysql', 'redis']) {
      expect(serviceBlock(name)).not.toMatch(/^    ports:/m)
      expect(serviceBlock(name)).toContain('- private')
    }
    expect(serviceBlock('frontend')).toContain('NUXT_PUBLIC_SITE_URL: ${NUXT_PUBLIC_SITE_URL:?NUXT_PUBLIC_SITE_URL is required}')
    expect(serviceBlock('frontend')).toContain('read_only: true')
    expect(serviceBlock('frontend')).toContain('no-new-privileges:true')
    expect(compose).toMatch(/^  private:\n    driver: bridge\n    internal: true/m)
  })

  it('镜像和公开前端配置只接收公开变量，示例文件不包含真实机密', async () => {
    const frontendDockerfile = await readRepositoryFile('frontend/Dockerfile')
    const backendDockerfile = await readRepositoryFile('backend/Dockerfile')
    const frontendDockerignore = await readRepositoryFile('frontend/.dockerignore')
    const backendDockerignore = await readRepositoryFile('backend/.dockerignore')
    const exampleEnvironment = await readRepositoryFile('deploy/1panel/env.production.example')
    const verifier = await readRepositoryFile('frontend/scripts/verify-production.mjs')

    for (const dockerfile of [frontendDockerfile, backendDockerfile]) {
      expect(dockerfile).not.toMatch(/(?:ARG|ENV)\s+(?:MYSQL_DSN|REDIS_PASSWORD|ADMIN_PASSWORD|QINIU_SECRET_KEY)/)
    }
    expect(frontendDockerfile).toContain('USER nuxt')
    expect(frontendDockerignore).toContain('.env*')
    expect(backendDockerignore).toContain('.env*')
    expect(verifier).toContain("resolve(import.meta.dirname, '../.output')")
    expect(verifier).toContain('loadEnvFile(environmentFile)')
    for (const secretName of ['MYSQL_DSN', 'MYSQL_PASSWORD', 'MYSQL_ROOT_PASSWORD', 'REDIS_PASSWORD', 'ADMIN_PASSWORD', 'QINIU_ACCESS_KEY', 'QINIU_SECRET_KEY']) {
      expect(verifier).toContain(`'${secretName}'`)
    }
    expect(verifier).toContain('process.env[name]')
    expect(exampleEnvironment).toContain('NUXT_PUBLIC_API_BASE=https://kagari-api.ykagari.top')
    expect(exampleEnvironment).toContain('NUXT_PUBLIC_SITE_URL=https://ykagari.top')
    expect(exampleEnvironment).not.toMatch(/(?:password|secret|access[_-]?key)=(?!replace-with-)/i)
  })
})
