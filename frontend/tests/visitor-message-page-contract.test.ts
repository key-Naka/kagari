import { describe, expect, it } from 'vitest'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

async function readFrontendFile(path: string): Promise<string> {
  return readFile(resolve(process.cwd(), path), 'utf8')
}

describe('Visitor Message 页面契约', () => {
  it('公开页面通过 SSR API 展示留言并提交匿名或昵称讯号', async () => {
    const source = await readFrontendFile('pages/visitor-messages/index.vue')

    expect(source).toContain('<script setup lang="ts">')
    expect(source).toContain('useFetch<PublicVisitorMessage[]>')
    expect(source).toContain('/api/v1/visitor-messages')
    expect(source).toContain('nickname')
    expect(source).toContain('email')
    expect(source).toContain('content')
    expect(source).toContain('maxlength="80"')
    expect(source).toContain('maxlength="254"')
    expect(source).toContain('maxlength="1000"')
    expect(source).toContain('邮箱不会公开')
    expect(source).toContain('IP 与当前提交路由')
    expect(source).not.toContain('v-html')
  })

  it('Administration Console 提供受保护的私有邮箱查看和永久删除入口', async () => {
    const [page, consolePage] = await Promise.all([
      readFrontendFile('pages/admin/visitor-messages.vue'),
      readFrontendFile('pages/admin/index.vue'),
    ])

    expect(page).toContain("definePageMeta({ middleware: 'admin-auth' })")
    expect(page).toContain('/api/v1/admin/visitor-messages')
    expect(page).toContain("method: 'DELETE'")
    expect(page).toContain("credentials: 'include'")
    expect(page).toContain('永久删除')
    expect(consolePage).toContain('to="/admin/visitor-messages"')
  })
})
