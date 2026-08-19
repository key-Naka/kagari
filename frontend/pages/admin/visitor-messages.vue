<script setup lang="ts">
import { computed, onMounted, shallowRef } from 'vue'

definePageMeta({ middleware: 'admin-auth' })

interface AdminVisitorMessage {
  id: number
  nickname: string
  email: string
  content: string
  createdAt: string
}

type ApiErrorPayload = { error?: string }

class ApiError extends Error {
  constructor(readonly status: number, message: string) {
    super(message)
  }
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const messages = shallowRef<AdminVisitorMessage[]>([])
const isLoading = shallowRef(true)
const deletingId = shallowRef<number | null>(null)
const errorMessage = shallowRef('')
const successMessage = shallowRef('')
const privateEmailCount = computed(() => messages.value.filter(message => message.email).length)

async function responseError(response: Response): Promise<string> {
  const fallback = `请求失败（HTTP ${response.status}）。`
  try {
    const payload = await response.json() as ApiErrorPayload
    return payload.error || fallback
  }
  catch {
    return fallback
  }
}

async function requestApi(path: string, options: RequestInit = {}): Promise<Response> {
  const response = await fetch(`${apiBase}${path}`, { ...options, credentials: 'include' })
  if (!response.ok) throw new ApiError(response.status, await responseError(response))
  return response
}

async function loadMessages(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''
  try {
    const response = await requestApi('/api/v1/admin/visitor-messages')
    messages.value = await response.json() as AdminVisitorMessage[]
  }
  catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await navigateTo('/admin/login?reason=session-expired')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '无法读取 Visitor Message。'
  }
  finally {
    isLoading.value = false
  }
}

async function deleteMessage(message: AdminVisitorMessage): Promise<void> {
  const author = message.nickname || '匿名访客'
  if (!window.confirm(`永久删除 ${author} 的这条 Visitor Message？此操作无法恢复。`)) return

  deletingId.value = message.id
  errorMessage.value = ''
  successMessage.value = ''
  try {
    await requestApi(`/api/v1/admin/visitor-messages/${message.id}`, { method: 'DELETE' })
    messages.value = messages.value.filter(candidate => candidate.id !== message.id)
    successMessage.value = 'Visitor Message 已永久删除。'
  }
  catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await navigateTo('/admin/login?reason=session-expired')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '删除 Visitor Message 失败。'
  }
  finally {
    deletingId.value = null
  }
}

function displayDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: 'Asia/Shanghai',
  }).format(new Date(value))
}

onMounted(loadMessages)
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-4 py-10 text-zinc-100 sm:px-6 lg:px-8">
    <section class="mx-auto max-w-6xl">
      <NuxtLink to="/admin" class="text-sm text-zinc-500 transition hover:text-zinc-200">返回管理控制台</NuxtLink>

      <header class="mt-8 flex flex-wrap items-end justify-between gap-5 border-b border-zinc-800 pb-7">
        <div>
          <p class="text-xs font-medium uppercase tracking-[0.24em] text-violet-300">Administration / Visitor Messages</p>
          <h1 class="mt-3 text-3xl font-semibold">访客留言管理</h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-zinc-400">邮箱只在此受保护页面显示。删除是永久操作，不存在回收站或恢复流程。</p>
        </div>
        <div class="grid grid-cols-2 gap-px border border-zinc-800 bg-zinc-800 text-center">
          <div class="bg-zinc-950 px-5 py-3"><strong class="block text-xl">{{ messages.length }}</strong><span class="text-xs text-zinc-500">全部讯号</span></div>
          <div class="bg-zinc-950 px-5 py-3"><strong class="block text-xl">{{ privateEmailCount }}</strong><span class="text-xs text-zinc-500">私有邮箱</span></div>
        </div>
      </header>

      <p v-if="errorMessage" class="mt-6 border border-rose-400/40 bg-rose-400/10 p-4 text-sm text-rose-100" role="alert">{{ errorMessage }}</p>
      <p v-if="successMessage" class="mt-6 border border-emerald-400/40 bg-emerald-400/10 p-4 text-sm text-emerald-100" role="status">{{ successMessage }}</p>
      <p v-if="isLoading" class="mt-8 border-b border-zinc-800 py-8 text-sm text-zinc-400" role="status">正在读取 Visitor Message。</p>
      <p v-else-if="messages.length === 0" class="mt-8 border-b border-zinc-800 py-8 text-sm text-zinc-400">当前没有 Visitor Message。</p>
      <ol v-else class="mt-8 divide-y divide-zinc-800 border-y border-zinc-800">
        <li v-for="message in messages" :key="message.id" class="grid gap-5 py-6 md:grid-cols-[12rem_minmax(0,1fr)_auto] md:items-start">
          <div>
            <p class="text-sm font-medium text-zinc-200">{{ message.nickname || '匿名访客' }}</p>
            <time :datetime="message.createdAt" class="mt-1 block font-mono text-xs text-zinc-600">{{ displayDate(message.createdAt) }}</time>
            <a v-if="message.email" :href="`mailto:${message.email}`" class="mt-3 block break-all text-xs text-violet-300 hover:text-violet-200">{{ message.email }}</a>
            <span v-else class="mt-3 block text-xs text-zinc-600">未留下邮箱</span>
          </div>
          <p class="whitespace-pre-wrap text-sm leading-7 text-zinc-300">{{ message.content }}</p>
          <button type="button" class="border border-rose-400/40 px-3 py-2 text-sm text-rose-200 transition hover:border-rose-200 disabled:cursor-wait disabled:opacity-50" :disabled="deletingId === message.id" @click="deleteMessage(message)">{{ deletingId === message.id ? '删除中' : '永久删除' }}</button>
        </li>
      </ol>
    </section>
  </main>
</template>
