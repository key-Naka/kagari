<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

definePageMeta({ middleware: 'admin-auth' })

type SiteConfig = Record<string, unknown>
type ApiErrorPayload = { error?: string }

class ApiError extends Error {
  constructor(readonly status: number, message: string) {
    super(message)
  }
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const configurationText = ref('{}')
const isLoadingConfig = ref(true)
const isSaving = ref(false)
const isLoggingOut = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const isBusy = computed(() => isLoadingConfig.value || isSaving.value || isLoggingOut.value)

async function getResponseError(response: Response): Promise<string> {
  const fallbackMessage = `请求失败（HTTP ${response.status}）`

  try {
    const payload = await response.json() as ApiErrorPayload
    return payload.error || fallbackMessage
  } catch {
    return fallbackMessage
  }
}

async function requestApi(path: string, options: RequestInit = {}): Promise<Response> {
  try {
    const response = await fetch(`${apiBase}${path}`, { ...options, credentials: 'include' })
    if (!response.ok) throw new ApiError(response.status, await getResponseError(response))
    return response
  } catch (error) {
    if (error instanceof Error) throw error
    throw new Error('网络请求失败，请检查网络连接后重试。')
  }
}

async function loadSiteConfig(): Promise<void> {
  isLoadingConfig.value = true
  errorMessage.value = ''

  try {
    const response = await requestApi('/api/v1/admin/site-config')
    configurationText.value = JSON.stringify(await response.json() as SiteConfig, null, 2)
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await navigateTo('/admin/login?reason=session-expired')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '无法加载站点配置。'
  } finally {
    isLoadingConfig.value = false
  }
}

async function saveSiteConfig(): Promise<void> {
  errorMessage.value = ''
  successMessage.value = ''

  let configuration: SiteConfig
  try {
    const parsed: unknown = JSON.parse(configurationText.value)
    if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') throw new Error('站点配置必须是 JSON 对象。')
    configuration = parsed as SiteConfig
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '站点配置不是有效的 JSON。'
    return
  }

  isSaving.value = true
  try {
    const response = await requestApi('/api/v1/admin/site-config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(configuration),
    })
    configurationText.value = JSON.stringify(await response.json() as SiteConfig, null, 2)
    successMessage.value = '站点配置已保存。'
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await navigateTo('/admin/login?reason=session-expired')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '保存失败，请稍后重试。'
  } finally {
    isSaving.value = false
  }
}

async function logout(): Promise<void> {
  errorMessage.value = ''
  successMessage.value = ''
  isLoggingOut.value = true

  try {
    await requestApi('/api/v1/admin/session', { method: 'DELETE' })
    await navigateTo('/admin/login?reason=logged-out')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '退出失败，请稍后重试。'
  } finally {
    isLoggingOut.value = false
  }
}

onMounted(loadSiteConfig)
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-4 py-10 text-zinc-100 sm:px-6 lg:px-8">
    <section class="mx-auto max-w-4xl">
      <header class="mb-8 flex flex-wrap items-end justify-between gap-4 border-b border-zinc-800 pb-6">
        <div>
          <p class="text-sm font-medium tracking-[0.2em] text-emerald-400">KAGARI</p>
          <h1 class="mt-2 text-3xl font-semibold tracking-tight">管理控制台</h1>
          <p class="mt-2 text-sm text-zinc-400">通过受保护的跨子域会话维护站点配置。</p>
        </div>
        <button type="button" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-500 hover:bg-zinc-900 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-50" :disabled="isBusy" @click="logout">
          {{ isLoggingOut ? '退出中…' : '退出登录' }}
        </button>
      </header>

      <nav class="mb-8 flex flex-wrap gap-3 border-b border-zinc-800 pb-6" aria-label="内容管理">
        <NuxtLink to="/admin/projects" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300">管理作品</NuxtLink>
        <NuxtLink to="/admin/posts" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300">管理博客</NuxtLink>
      </nav>

      <div v-if="isLoadingConfig" class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-6 text-sm text-zinc-300" role="status">正在加载站点配置…</div>
      <template v-else>
        <p v-if="errorMessage" class="mb-5 rounded-md border border-red-900/70 bg-red-950/50 px-4 py-3 text-sm text-red-200" role="alert">{{ errorMessage }}</p>
        <p v-if="successMessage" class="mb-5 rounded-md border border-emerald-900/70 bg-emerald-950/50 px-4 py-3 text-sm text-emerald-200" role="status">{{ successMessage }}</p>
        <form class="rounded-xl border border-zinc-800 bg-zinc-900/40 p-6 shadow-2xl shadow-black/20" @submit.prevent="saveSiteConfig">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="text-lg font-semibold">站点配置</h2>
              <p class="mt-1 text-sm text-zinc-400">仅接受 JSON 对象；保存前会在浏览器中完成语法与类型校验。</p>
            </div>
            <span class="rounded-full bg-emerald-400/10 px-3 py-1 text-xs font-medium text-emerald-300">会话已认证</span>
          </div>
          <label class="mt-6 block text-sm font-medium text-zinc-200" for="site-config">配置 JSON</label>
          <textarea id="site-config" v-model="configurationText" rows="18" spellcheck="false" aria-describedby="site-config-hint" class="mt-2 block w-full resize-y rounded-md border border-zinc-700 bg-zinc-950 p-4 font-mono text-sm leading-6 text-zinc-100 outline-none transition focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400/20" :disabled="isSaving" />
          <p id="site-config-hint" class="mt-2 text-xs text-zinc-500">更新将立即写入服务端站点配置。</p>
          <button type="submit" class="mt-6 rounded-md bg-emerald-400 px-5 py-2.5 text-sm font-semibold text-zinc-950 transition hover:bg-emerald-300 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-50" :disabled="isSaving">{{ isSaving ? '保存中…' : '保存配置' }}</button>
        </form>
      </template>
    </section>
  </main>
</template>
