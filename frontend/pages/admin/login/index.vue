<script setup lang="ts">
import { computed, ref } from 'vue'

definePageMeta({
  middleware: 'admin-auth',
})

type ApiErrorPayload = {
  error?: string
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const route = useRoute()

const username = ref('')
const password = ref('')
const isSubmitting = ref(false)
const errorMessage = ref('')
const statusMessage = computed(() => {
  if (route.query.reason === 'session-expired') {
    return '管理会话已过期，请重新登录。'
  }

  if (route.query.reason === 'logged-out') {
    return '已安全退出。'
  }

  return ''
})

/** 将 API 响应转换为可展示的错误信息。 */
async function getResponseError(response: Response): Promise<string> {
  const fallbackMessage = `登录失败（HTTP ${response.status}）`

  try {
    const payload = await response.json() as ApiErrorPayload
    return payload.error || fallbackMessage
  } catch {
    return fallbackMessage
  }
}

/** 使用账号密码创建管理会话，成功后进入受保护的控制台。 */
async function login(): Promise<void> {
  errorMessage.value = ''
  isSubmitting.value = true

  try {
    const response = await fetch(`${apiBase}/api/v1/admin/session`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value.trim(), password: password.value }),
      credentials: 'include',
    })

    if (!response.ok) {
      throw new Error(await getResponseError(response))
    }

    password.value = ''
    await navigateTo('/admin')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '登录失败，请稍后重试。'
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-4 py-10 text-zinc-100 sm:px-6 lg:px-8">
    <section class="mx-auto max-w-md">
      <header class="mb-8 border-b border-zinc-800 pb-6">
        <p class="text-sm font-medium tracking-[0.2em] text-emerald-400">KAGARI</p>
        <h1 class="mt-2 text-3xl font-semibold tracking-tight">管理员登录</h1>
        <p class="mt-2 text-sm text-zinc-400">请输入部署环境中配置的管理员凭据。</p>
      </header>

      <p v-if="errorMessage" class="mb-5 rounded-md border border-red-900/70 bg-red-950/50 px-4 py-3 text-sm text-red-200" role="alert">
        {{ errorMessage }}
      </p>
      <p v-else-if="statusMessage" class="mb-5 rounded-md border border-emerald-900/70 bg-emerald-950/50 px-4 py-3 text-sm text-emerald-200" role="status">
        {{ statusMessage }}
      </p>

      <form class="rounded-xl border border-zinc-800 bg-zinc-900/40 p-6 shadow-2xl shadow-black/20" @submit.prevent="login">
        <div class="space-y-4">
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
    </section>
  </main>
</template>
