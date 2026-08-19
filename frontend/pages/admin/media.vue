<script setup lang="ts">
import { onMounted, shallowRef } from 'vue'
import type { AdminMedia } from '~/composables/useAdminMediaUpload'

definePageMeta({ middleware: 'admin-auth' })

const { requestApi, redirectExpiredSession } = useAdminApi()
const { uploadMedia } = useAdminMediaUpload()
const media = shallowRef<AdminMedia[]>([])
const selectedFile = shallowRef<File | null>(null)
const fileInputKey = shallowRef(0)
const isLoading = shallowRef(true)
const isUploading = shallowRef(false)
const uploadStage = shallowRef('')
const errorMessage = shallowRef('')
const successMessage = shallowRef('')

async function loadMedia(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''
  try {
    const response = await requestApi('/api/v1/admin/media')
    media.value = await response.json() as AdminMedia[]
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '无法读取媒体。'
  } finally {
    isLoading.value = false
  }
}

async function uploadSelected(): Promise<void> {
  if (!selectedFile.value) return
  isUploading.value = true
  errorMessage.value = ''
  successMessage.value = ''
  try {
    const kind = selectedFile.value.type.startsWith('audio/') ? 'audio' : 'image'
    await uploadMedia(selectedFile.value, kind, stage => { uploadStage.value = stage })
    selectedFile.value = null
    fileInputKey.value += 1
    await loadMedia()
    successMessage.value = '媒体已上传并登记，可复制公开地址用于作品、博客或分享图。'
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '上传媒体失败。'
  } finally {
    isUploading.value = false
    uploadStage.value = ''
  }
}

async function copyUrl(url: string): Promise<void> {
  await navigator.clipboard.writeText(url)
  successMessage.value = '公开地址已复制。'
}

onMounted(loadMedia)
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-4 py-10 text-zinc-100 sm:px-6 lg:px-8">
    <section class="mx-auto max-w-6xl">
      <NuxtLink to="/admin" class="text-sm text-zinc-500 transition hover:text-zinc-200">返回管理控制台</NuxtLink>
      <header class="mt-8 border-b border-zinc-800 pb-7">
        <p class="text-xs font-medium uppercase tracking-[0.24em] text-violet-400">Administration / Media</p>
        <h1 class="mt-3 text-3xl font-semibold">媒体库</h1>
        <p class="mt-2 max-w-2xl text-sm leading-6 text-zinc-400">图片与音频使用短期凭证直接上传七牛；前端不会接触存储密钥。</p>
      </header>

      <p v-if="errorMessage" class="mt-6 border border-rose-400/40 bg-rose-400/10 p-4 text-sm text-rose-100" role="alert">{{ errorMessage }}</p>
      <p v-if="successMessage" class="mt-6 border border-emerald-400/40 bg-emerald-400/10 p-4 text-sm text-emerald-100" role="status">{{ successMessage }}</p>
      <p v-if="uploadStage" class="mt-6 border border-violet-400/40 bg-violet-400/10 p-4 text-sm text-violet-100" role="status">{{ uploadStage }}</p>

      <form class="mt-8 flex flex-wrap items-end gap-4 border border-zinc-800 bg-zinc-900/30 p-5" @submit.prevent="uploadSelected">
        <label class="min-w-64 flex-1 text-sm font-medium">图片或音频文件<input :key="fileInputKey" required type="file" accept="image/jpeg,image/png,image/webp,image/avif,audio/mpeg,audio/ogg,audio/wav" class="mt-2 block w-full text-sm text-zinc-400 file:mr-4 file:border-0 file:bg-zinc-800 file:px-4 file:py-2 file:text-zinc-100" @change="selectedFile = ($event.currentTarget as HTMLInputElement).files?.[0] ?? null" /></label>
        <button type="submit" :disabled="isUploading || !selectedFile" class="bg-violet-300 px-5 py-2.5 text-sm font-semibold text-zinc-950 disabled:opacity-50">{{ isUploading ? '上传中…' : '上传媒体' }}</button>
      </form>

      <p v-if="isLoading" class="py-8 text-sm text-zinc-400" role="status">正在读取媒体…</p>
      <p v-else-if="media.length === 0" class="py-8 text-sm text-zinc-400">媒体库为空。上传第一张图片或音频。</p>
      <ul v-else class="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <li v-for="item in media" :key="item.id" class="overflow-hidden border border-zinc-800 bg-zinc-900/30">
          <img v-if="item.kind === 'image'" :src="item.publicUrl" :alt="item.originalName" class="aspect-video w-full object-cover" />
          <div v-else class="flex aspect-video items-center justify-center bg-violet-950/30 text-4xl" aria-hidden="true">♫</div>
          <div class="p-4">
            <p class="truncate text-sm font-medium">{{ item.originalName }}</p>
            <p class="mt-1 text-xs text-zinc-500">{{ item.kind === 'image' ? `${item.width} × ${item.height}` : `${Math.round((item.durationMs ?? 0) / 1000)} 秒` }}</p>
            <button type="button" class="mt-4 border border-zinc-700 px-3 py-2 text-xs hover:border-zinc-300" @click="copyUrl(item.publicUrl)">复制公开地址</button>
          </div>
        </li>
      </ul>
    </section>
  </main>
</template>
