<script setup lang="ts">
import { computed, onMounted, ref, shallowRef } from 'vue'
import type { TrackMedia } from '~/stores/player'

definePageMeta({ middleware: 'admin-auth' })

interface AdminTrack {
  id: number
  title: string
  cover: TrackMedia
  audio: TrackMedia
  enabled: boolean
  sortOrder: number
}

interface TrackForm {
  title: string
  enabled: boolean
  sortOrder: number
}

interface UploadCredentials {
  uploadToken: string
  uploadUrl: string
  objectKey: string
}

interface DetectedMetadata {
  width: number
  height: number
  durationMs: number
}

type ApiErrorPayload = { error?: string }

class ApiError extends Error {
  constructor(readonly status: number, message: string) {
    super(message)
  }
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const tracks = shallowRef<AdminTrack[]>([])
const activeTrackId = shallowRef<number | null>(null)
const coverFile = shallowRef<File | null>(null)
const audioFile = shallowRef<File | null>(null)
const form = ref<TrackForm>(newTrackForm())
const isLoading = shallowRef(true)
const isSaving = shallowRef(false)
const uploadStage = shallowRef('')
const errorMessage = shallowRef('')
const successMessage = shallowRef('')

const isEditing = computed(() => activeTrackId.value !== null)
const activeTrack = computed(() => tracks.value.find(track => track.id === activeTrackId.value) ?? null)

function newTrackForm(): TrackForm {
  return { title: '', enabled: false, sortOrder: 0 }
}

async function responseError(response: Response): Promise<string> {
  const fallback = `请求失败（HTTP ${response.status}）。`
  try {
    const payload = await response.json() as ApiErrorPayload
    return payload.error || fallback
  } catch {
    return fallback
  }
}

async function requestApi(path: string, options: RequestInit = {}): Promise<Response> {
  try {
    const response = await fetch(`${apiBase}${path}`, { ...options, credentials: 'include' })
    if (!response.ok) throw new ApiError(response.status, await responseError(response))
    return response
  } catch (error) {
    if (error instanceof Error) throw error
    throw new Error('网络请求失败，请检查连接后重试。')
  }
}

async function loadTracks(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''
  try {
    const response = await requestApi('/api/v1/admin/tracks')
    tracks.value = await response.json() as AdminTrack[]
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await navigateTo('/admin/login?reason=session-expired')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '无法读取 Track。'
  } finally {
    isLoading.value = false
  }
}

function startCreate(): void {
  activeTrackId.value = null
  form.value = newTrackForm()
  coverFile.value = null
  audioFile.value = null
  errorMessage.value = ''
  successMessage.value = ''
}

function startEdit(track: AdminTrack): void {
  activeTrackId.value = track.id
  form.value = { title: track.title, enabled: track.enabled, sortOrder: track.sortOrder }
  coverFile.value = null
  audioFile.value = null
  errorMessage.value = ''
  successMessage.value = ''
}

function selectCover(event: Event): void {
  coverFile.value = (event.currentTarget as HTMLInputElement).files?.[0] ?? null
}

function selectAudio(event: Event): void {
  audioFile.value = (event.currentTarget as HTMLInputElement).files?.[0] ?? null
}

async function detectImageMetadata(file: File): Promise<DetectedMetadata> {
  const objectUrl = URL.createObjectURL(file)
  const image = new Image()
  try {
    return await new Promise((resolve, reject) => {
      image.addEventListener('load', () => {
        if (image.naturalWidth <= 0 || image.naturalHeight <= 0) {
          reject(new Error('无法识别封面尺寸。'))
          return
        }
        resolve({ width: image.naturalWidth, height: image.naturalHeight, durationMs: 0 })
      }, { once: true })
      image.addEventListener('error', () => reject(new Error('无法读取封面文件。')), { once: true })
      image.src = objectUrl
    })
  } finally {
    URL.revokeObjectURL(objectUrl)
  }
}

async function detectAudioMetadata(file: File): Promise<DetectedMetadata> {
  const objectUrl = URL.createObjectURL(file)
  const audio = new Audio()
  audio.preload = 'metadata'
  try {
    return await new Promise((resolve, reject) => {
      audio.addEventListener('loadedmetadata', () => {
        if (!Number.isFinite(audio.duration) || audio.duration <= 0) {
          reject(new Error('无法识别音频时长。'))
          return
        }
        resolve({ width: 0, height: 0, durationMs: Math.round(audio.duration * 1000) })
      }, { once: true })
      audio.addEventListener('error', () => reject(new Error('无法读取音频媒体元数据。')), { once: true })
      audio.src = objectUrl
      audio.load()
    })
  } finally {
    audio.removeAttribute('src')
    audio.load()
    URL.revokeObjectURL(objectUrl)
  }
}

async function uploadMedia(file: File, kind: 'image' | 'audio'): Promise<TrackMedia> {
  const metadata = kind === 'image' ? await detectImageMetadata(file) : await detectAudioMetadata(file)
  const credentialsResponse = await requestApi('/api/v1/admin/media/upload-credentials', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ kind, mimeType: file.type, size: file.size, filename: file.name }),
  })
  const credentials = await credentialsResponse.json() as UploadCredentials

  const upload = new FormData()
  upload.append('token', credentials.uploadToken)
  upload.append('key', credentials.objectKey)
  upload.append('file', file)
  const uploadResponse = await fetch(credentials.uploadUrl, { method: 'POST', body: upload })
  if (!uploadResponse.ok) throw new Error(`媒体上传失败（HTTP ${uploadResponse.status}）。`)

  const registrationResponse = await requestApi('/api/v1/admin/media', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      objectKey: credentials.objectKey,
      kind,
      mimeType: file.type,
      size: file.size,
      originalName: file.name,
      ...metadata,
    }),
  })
  return await registrationResponse.json() as TrackMedia
}

async function saveTrack(): Promise<void> {
  errorMessage.value = ''
  successMessage.value = ''
  const existing = activeTrack.value
  if (!existing && (!coverFile.value || !audioFile.value)) {
    errorMessage.value = '新建 Track 必须选择封面与音频文件。'
    return
  }

  isSaving.value = true
  try {
    let cover = existing?.cover ?? null
    let audio = existing?.audio ?? null
    if (coverFile.value) {
      uploadStage.value = '正在上传封面……'
      cover = await uploadMedia(coverFile.value, 'image')
    }
    if (audioFile.value) {
      uploadStage.value = '正在识别时长并上传音频……'
      audio = await uploadMedia(audioFile.value, 'audio')
    }
    if (!cover || !audio) throw new Error('Track 的封面或音频媒体缺失。')

    uploadStage.value = '正在保存 Track……'
    const path = activeTrackId.value === null ? '/api/v1/admin/tracks' : `/api/v1/admin/tracks/${activeTrackId.value}`
    await requestApi(path, {
      method: activeTrackId.value === null ? 'POST' : 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: form.value.title, coverMediaId: cover.id, audioMediaId: audio.id, enabled: form.value.enabled, sortOrder: form.value.sortOrder }),
    })

    const wasCreating = activeTrackId.value === null
    await loadTracks()
    startCreate()
    successMessage.value = wasCreating ? 'Track 已创建。' : 'Track 已更新。'
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await navigateTo('/admin/login?reason=session-expired')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '保存 Track 失败。'
  } finally {
    uploadStage.value = ''
    isSaving.value = false
  }
}

onMounted(loadTracks)
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-4 py-10 text-zinc-100 sm:px-6 lg:px-8">
    <section class="mx-auto max-w-6xl">
      <NuxtLink to="/admin" class="text-sm text-zinc-500 transition hover:text-zinc-200">返回管理控制台</NuxtLink>
      <header class="mt-8 flex flex-wrap items-end justify-between gap-5 border-b border-zinc-800 pb-7">
        <div><p class="text-xs font-medium uppercase tracking-[0.24em] text-violet-400">Administration / Resonance</p><h1 class="mt-3 text-3xl font-semibold">Track 管理</h1><p class="mt-2 max-w-2xl text-sm leading-6 text-zinc-400">上传受限媒体、自动记录音频时长，并控制公开音乐档案中的启用状态与顺序。</p></div>
        <button type="button" class="border border-zinc-700 px-4 py-2 text-sm transition hover:border-zinc-300 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300" @click="startCreate">新建 Track</button>
      </header>

      <p v-if="errorMessage" class="mt-6 border border-rose-400/40 bg-rose-400/10 p-4 text-sm text-rose-100" role="alert">{{ errorMessage }}</p>
      <p v-if="successMessage" class="mt-6 border border-emerald-400/40 bg-emerald-400/10 p-4 text-sm text-emerald-100" role="status">{{ successMessage }}</p>
      <p v-if="uploadStage" class="mt-6 border border-violet-400/40 bg-violet-400/10 p-4 text-sm text-violet-100" role="status">{{ uploadStage }}</p>

      <div class="mt-8 grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(23rem,0.82fr)]">
        <section aria-labelledby="track-list-heading">
          <div class="flex items-baseline justify-between border-b border-zinc-800 pb-3"><h2 id="track-list-heading" class="text-lg font-semibold">全部 Track</h2><span class="text-sm text-zinc-500">{{ tracks.length }} 条</span></div>
          <p v-if="isLoading" class="py-6 text-sm text-zinc-400" role="status">正在读取 Track…</p>
          <p v-else-if="tracks.length === 0" class="border-b border-zinc-800 py-8 text-sm text-zinc-400">还没有 Track。先上传封面和音频。</p>
          <ol v-else class="divide-y divide-zinc-800">
            <li v-for="track in tracks" :key="track.id" class="py-4">
              <button type="button" class="grid w-full grid-cols-[4rem_minmax(0,1fr)_auto] items-center gap-4 text-left hover:bg-zinc-900/50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300" @click="startEdit(track)">
                <img :src="track.cover.publicUrl" :alt="`${track.title} 封面`" width="64" height="64" class="aspect-square w-16 object-cover">
                <span class="min-w-0"><strong class="block truncate font-medium">{{ track.title }}</strong><small class="mt-1 block truncate text-zinc-500">{{ track.audio.originalName }} · {{ Math.round(track.audio.durationMs / 1000) }}s</small></span>
                <span class="text-right"><small class="block text-zinc-500">#{{ track.sortOrder }}</small><small :class="track.enabled ? 'text-emerald-300' : 'text-zinc-500'">{{ track.enabled ? '已启用' : '已停用' }}</small></span>
              </button>
            </li>
          </ol>
        </section>

        <section class="border border-zinc-800 bg-zinc-900/30 p-5 sm:p-6" aria-labelledby="track-form-heading">
          <p class="text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">{{ isEditing ? 'Edit signal' : 'New signal' }}</p><h2 id="track-form-heading" class="mt-2 text-xl font-semibold">{{ isEditing ? '编辑 Track' : '新建 Track' }}</h2>
          <form class="mt-6 space-y-5" @submit.prevent="saveTrack">
            <label class="block text-sm font-medium">标题<input v-model.trim="form.title" name="track-title" autocomplete="off" required maxlength="160" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300"></label>
            <div class="grid gap-5 sm:grid-cols-2"><label class="block text-sm font-medium">排序<input v-model.number="form.sortOrder" name="track-sort-order" autocomplete="off" inputmode="numeric" required min="0" type="number" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300"></label><label class="flex items-end gap-3 pb-2 text-sm font-medium"><input v-model="form.enabled" name="track-enabled" type="checkbox" class="h-4 w-4 accent-violet-400 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300">公开启用</label></div>
            <label class="block text-sm font-medium">封面文件<input name="track-cover" type="file" accept="image/jpeg,image/png,image/webp,image/avif" :required="!isEditing" class="mt-2 block w-full text-sm text-zinc-400 file:mr-4 file:border-0 file:bg-zinc-800 file:px-3 file:py-2 file:text-zinc-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300" @change="selectCover"><small v-if="activeTrack" class="mt-2 block break-words text-zinc-500">当前：{{ activeTrack.cover.originalName }}（选择新文件即可替换）</small></label>
            <label class="block text-sm font-medium">音频文件<input name="track-audio" type="file" accept="audio/mpeg,audio/ogg,audio/wav" :required="!isEditing" class="mt-2 block w-full text-sm text-zinc-400 file:mr-4 file:border-0 file:bg-zinc-800 file:px-3 file:py-2 file:text-zinc-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300" @change="selectAudio"><small v-if="activeTrack" class="mt-2 block break-words text-zinc-500">当前：{{ activeTrack.audio.originalName }} · {{ Math.round(activeTrack.audio.durationMs / 1000) }} 秒</small><small class="mt-2 block text-zinc-500">时长会从浏览器媒体元数据自动识别，不能手工填写。</small></label>
            <div class="flex flex-wrap gap-3 border-t border-zinc-800 pt-5"><button type="submit" class="bg-violet-200 px-4 py-2 text-sm font-semibold text-zinc-950 hover:bg-violet-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300 disabled:cursor-wait disabled:opacity-50" :disabled="isSaving">{{ isSaving ? '处理中…' : isEditing ? '保存修改' : '创建 Track' }}</button><button v-if="isEditing" type="button" class="border border-zinc-700 px-4 py-2 text-sm hover:border-zinc-300 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-violet-300" :disabled="isSaving" @click="startCreate">取消编辑</button></div>
          </form>
        </section>
      </div>
    </section>
  </main>
</template>
