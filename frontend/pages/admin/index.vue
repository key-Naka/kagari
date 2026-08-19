<script setup lang="ts">
import { computed, onMounted, ref, shallowRef } from 'vue'

definePageMeta({ middleware: 'admin-auth' })

interface SiteConfig {
  siteTitle: string
  seoSummary: string
  shareImageUrl: string
}

const { requestApi, redirectExpiredSession } = useAdminApi()
const { uploadMedia } = useAdminMediaUpload()
const configuration = ref<SiteConfig>({ siteTitle: '', seoSummary: '', shareImageUrl: '' })
const shareImageFile = shallowRef<File | null>(null)
const shareImageInputKey = shallowRef(0)
const uploadStage = shallowRef('')
const isLoadingConfig = ref(true)
const isSaving = ref(false)
const isLoggingOut = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const isBusy = computed(() => isLoadingConfig.value || isSaving.value || isLoggingOut.value)

async function loadSiteConfig(): Promise<void> {
  isLoadingConfig.value = true
  errorMessage.value = ''

  try {
    const response = await requestApi('/api/v1/admin/site-config')
    configuration.value = await response.json() as SiteConfig
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '无法加载站点配置。'
  } finally {
    isLoadingConfig.value = false
  }
}

async function saveSiteConfig(): Promise<void> {
  errorMessage.value = ''
  successMessage.value = ''
  isSaving.value = true
  try {
    if (shareImageFile.value) {
      const media = await uploadMedia(shareImageFile.value, 'image', stage => { uploadStage.value = stage })
      configuration.value.shareImageUrl = media.publicUrl
    }
    const response = await requestApi('/api/v1/admin/site-config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(configuration.value),
    })
    configuration.value = await response.json() as SiteConfig
    shareImageFile.value = null
    shareImageInputKey.value += 1
    successMessage.value = '站点配置已保存。'
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '保存失败，请稍后重试。'
  } finally {
    uploadStage.value = ''
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
          <p class="mt-2 text-sm text-zinc-400">通过受保护的跨子域会话维护内容、媒体与站点配置。</p>
        </div>
        <button type="button" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-500 hover:bg-zinc-900 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-50" :disabled="isBusy" @click="logout">
          {{ isLoggingOut ? '退出中…' : '退出登录' }}
        </button>
      </header>

      <nav class="mb-8 flex flex-wrap gap-3 border-b border-zinc-800 pb-6" aria-label="内容管理">
        <NuxtLink to="/admin/projects" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300">管理作品</NuxtLink>
        <NuxtLink to="/admin/posts" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300">管理博客</NuxtLink>
        <NuxtLink to="/admin/tracks" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300">管理 Track</NuxtLink>
        <NuxtLink to="/admin/gallery-items" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300">管理 Album Item</NuxtLink>
        <NuxtLink to="/admin/media" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300">管理媒体</NuxtLink>
        <NuxtLink to="/admin/visitor-messages" class="rounded-md border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300">管理 Visitor Message</NuxtLink>
      </nav>

      <div v-if="isLoadingConfig" class="rounded-lg border border-zinc-800 bg-zinc-900/40 p-6 text-sm text-zinc-300" role="status">正在加载站点配置…</div>
      <template v-else>
        <p v-if="errorMessage" class="mb-5 rounded-md border border-red-900/70 bg-red-950/50 px-4 py-3 text-sm text-red-200" role="alert">{{ errorMessage }}</p>
        <p v-if="successMessage" class="mb-5 rounded-md border border-emerald-900/70 bg-emerald-950/50 px-4 py-3 text-sm text-emerald-200" role="status">{{ successMessage }}</p>
        <p v-if="uploadStage" class="mb-5 rounded-md border border-violet-900/70 bg-violet-950/50 px-4 py-3 text-sm text-violet-200" role="status">{{ uploadStage }}</p>
        <form class="rounded-xl border border-zinc-800 bg-zinc-900/40 p-6 shadow-2xl shadow-black/20" @submit.prevent="saveSiteConfig">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 class="text-lg font-semibold">站点配置</h2>
              <p class="mt-1 text-sm text-zinc-400">维护首页标题、搜索摘要与社交分享图片；保存后立即作用于公开首页元数据。</p>
            </div>
            <span class="rounded-full bg-emerald-400/10 px-3 py-1 text-xs font-medium text-emerald-300">会话已认证</span>
          </div>
          <label class="mt-6 block text-sm font-medium text-zinc-200">站点标题<input v-model.trim="configuration.siteTitle" required maxlength="120" class="mt-2 block w-full rounded-md border border-zinc-700 bg-zinc-950 px-4 py-3 outline-none focus:border-emerald-400" /></label>
          <label class="mt-5 block text-sm font-medium text-zinc-200">SEO 摘要<textarea v-model.trim="configuration.seoSummary" required maxlength="300" rows="4" class="mt-2 block w-full resize-y rounded-md border border-zinc-700 bg-zinc-950 px-4 py-3 leading-6 outline-none focus:border-emerald-400" /></label>
          <label class="mt-5 block text-sm font-medium text-zinc-200">分享图片 HTTPS 地址<input v-model.trim="configuration.shareImageUrl" type="url" class="mt-2 block w-full rounded-md border border-zinc-700 bg-zinc-950 px-4 py-3 outline-none focus:border-emerald-400" /></label>
          <label class="mt-5 block text-sm font-medium text-zinc-200">或上传分享图片<input :key="shareImageInputKey" type="file" accept="image/jpeg,image/png,image/webp,image/avif" class="mt-2 block w-full text-sm text-zinc-400 file:mr-4 file:border-0 file:bg-zinc-800 file:px-4 file:py-2 file:text-zinc-100" @change="shareImageFile = ($event.currentTarget as HTMLInputElement).files?.[0] ?? null" /></label>
          <button type="submit" class="mt-6 rounded-md bg-emerald-400 px-5 py-2.5 text-sm font-semibold text-zinc-950 transition hover:bg-emerald-300 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-400 disabled:cursor-not-allowed disabled:opacity-50" :disabled="isSaving">{{ isSaving ? '保存中…' : '保存配置' }}</button>
        </form>
      </template>
    </section>
  </main>
</template>
