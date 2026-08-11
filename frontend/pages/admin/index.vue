<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

type SiteConfig = Record<string, unknown>

type ApiErrorPayload = {
  error?: string
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')

const username = ref('')
const password = ref('')
const configurationText = ref('{}')
const isAuthenticated = ref(false)
const isCheckingSession = ref(true)
const isSubmitting = ref(false)
const isSaving = ref(false)
const errorMessage = ref('')
const successMessage = ref('')

const isBusy = computed(() => isCheckingSession.value || isSubmitting.value || isSaving.value)

/** 将 API 响应转换为可展示的错误信息。 */
async function getResponseError(response: Response): Promise<string> {
  const fallbackMessage = `请求失败（HTTP ${response.status}）`

  try {
    const payload = await response.json() as ApiErrorPayload
    return payload.error || fallbackMessage
  } catch {
    return fallbackMessage
  }
}

/** 发起包含跨子域会话 Cookie 的 API 请求，并在失败时抛出错误。 */
async function requestApi(path: string, options: RequestInit = {}): Promise<Response> {
  try {
    const response = await fetch(`${apiBase}${path}`, {
      ...options,
      credentials: 'include',
    })

    if (!response.ok) {
      throw new Error(await getResponseError(response))
    }

    return response
  } catch (error) {
    if (error instanceof Error) {
      throw error
    }

    throw new Error('网络请求失败，请检查网络连接后重试。')
  }
}

/** 获取当前会话状态；未认证是预期状态，不显示错误。 */
async function checkSession(): Promise<void> {
  isCheckingSession.value = true
  errorMessage.value = ''
  successMessage.value = ''

  try {
    const response = await fetch(`${apiBase}/api/v1/admin/session`, { credentials: 'include' })

    if (response.status === 401) {
      isAuthenticated.value = false
      return
    }

    if (!response.ok) {
      throw new Error(await getResponseError(response))
    }

    isAuthenticated.value = true
    await loadSiteConfig()
  } catch (error) {
    isAuthenticated.value = false
    errorMessage.value = error instanceof Error ? error.message : '无法检查登录状态。'
  } finally {
    isCheckingSession.value = false
  }
}

/** 读取站点配置并格式化为可编辑 JSON。 */
async function loadSiteConfig(): Promise<void> {
  try {
    const response = await requestApi('/api/v1/admin/site-config')
    const configuration = await response.json() as SiteConfig
    configurationText.value = JSON.stringify(configuration, null, 2)
  } catch (error) {
    if (error instanceof Error && error.message.includes('HTTP 401')) {
      isAuthenticated.value = false
    }
    throw error
  }
}

/** 使用账号和密码创建管理会话，再加载当前站点配置。 */
async function login(): Promise<void> {
  errorMessage.value = ''
  successMessage.value = ''
  isSubmitting.value = true

  try {
    await requestApi('/api/v1/admin/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value, password: password.value }),
    })
    password.value = ''
    isAuthenticated.value = true
    await loadSiteConfig()
    successMessage.value = '登录成功，已加载当前站点配置。'
  } catch (error) {
    isAuthenticated.value = false
    errorMessage.value = error instanceof Error ? error.message : '登录失败，请稍后重试。'
  } finally {
    isSubmitting.value = false
  }
}

/** 校验 JSON 对象后保存站点配置。 */
async function saveSiteConfig(): Promise<void> {
  errorMessage.value = ''
  successMessage.value = ''

  let configuration: SiteConfig
  try {
    const parsed: unknown = JSON.parse(configurationText.value)
    if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
      throw new Error('站点配置必须是 JSON 对象。')
    }
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
    const savedConfiguration = await response.json() as SiteConfig
    configurationText.value = JSON.stringify(savedConfiguration, null, 2)
    successMessage.value = '站点配置已保存。'
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '保存失败，请稍后重试。'
  } finally {
    isSaving.value = false
  }
}

/** 删除服务端会话 Cookie 并切换至登录界面。 */
async function logout(): Promise<void> {
  errorMessage.value = ''
  successMessage.value = ''
  isSubmitting.value = true

  try {
    await requestApi('/api/v1/admin/session', { method: 'DELETE' })
    isAuthenticated.value = false
    username.value = ''
    password.value = ''
    configurationText.value = '{}'
    successMessage.value = '已安全退出。'
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '退出失败，请稍后重试。'
  } finally {
    isSubmitting.value = false
  }
}

onMounted(checkSession)
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
        <button
          v-if="isAuthenticated"
          type="button"
          class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-500 hover:bg-zinc-900 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="isBusy"
          @click="logout"
        >
          {{ isSubmitting ? '退出中…' : '退出登录' }}
        </button>
      </header>

      <div v-if="isCheckingSession" class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-6 text-sm text-zinc-300" role="status">
        正在验证管理会话…
      </div>

      <template v-else>
        <p v-if="errorMessage" class="mb-5 rounded-md border border-red-900/70 bg-red-950/50 px-4 py-3 text-sm text-red-200" role="alert">
          {{ errorMessage }}
        </p>
        <p v-if="successMessage" class="mb-5 rounded-md border border-emerald-900/70 bg-emerald-950/50 px-4 py-3 text-sm text-emerald-200" role="status">
          {{ successMessage }}
        </p>

        <form v-if="!isAuthenticated" class="max-w-md rounded-xl border border-zinc-800 bg-zinc-900/40 p-6 shadow-2xl shadow-black/20" @submit.prevent="login">
          <h2 class="text-lg font-semibold">管理员登录</h2>
          <p class="mt-1 text-sm text-zinc-400">请输入部署环境中配置的管理员凭据。</p>
          <div class="mt-6 space-y-4">
            <label class="block text-sm font-medium text-zinc-200" for="username">
              账号
              <input id="username" v-model.trim="username" autocomplete="username" required class="mt-2 w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100 outline-none transition placeholder:text-zinc-600 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400/20" :disabled="isSubmitting" />
            </label>
            <label class="block text-sm font-medium text-zinc-200" for="password">
              密码
              <input id="password" v-model="password" type="password" autocomplete="current-password" required class="mt-2 w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100 outline-none transition placeholder:text-zinc-600 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-400/20" :disabled="isSubmitting" />
            </label>
          </div>
          <button type="submit" class="mt-6 w-full rounded-md bg-emerald-400 px-4 py-2.5 text-sm font-semibold text-zinc-950 transition hover:bg-emerald-300 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-50" :disabled="isSubmitting">
            {{ isSubmitting ? '登录中…' : '登录' }}
          </button>
        </form>

        <form v-else class="rounded-xl border border-zinc-800 bg-zinc-900/40 p-6 shadow-2xl shadow-black/20" @submit.prevent="saveSiteConfig">
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
          <button type="submit" class="mt-6 rounded-md bg-emerald-400 px-5 py-2.5 text-sm font-semibold text-zinc-950 transition hover:bg-emerald-300 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-50" :disabled="isSaving">
            {{ isSaving ? '保存中…' : '保存配置' }}
          </button>
        </form>
      </template>
    </section>
  </main>
</template>
