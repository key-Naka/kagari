<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

definePageMeta({ middleware: 'admin-auth' })

type PostStatus = 'draft' | 'published' | 'archived'

interface AdminPost {
  id: number
  title: string
  slug: string
  summary: string
  shareImageUrl: string
  content: string
  tags: string[]
  status: PostStatus
  publishedAt: string
}

interface PostForm {
  title: string
  slug: string
  summary: string
  shareImageUrl: string
  content: string
  tags: string
  status: PostStatus
}

type ApiErrorPayload = { error?: string }

class ApiError extends Error {
  constructor(readonly status: number, message: string) {
    super(message)
  }
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const posts = ref<AdminPost[]>([])
const activePostId = ref<number | null>(null)
const form = ref<PostForm>(newPostForm())
const isLoading = ref(true)
const isSaving = ref(false)
const deletingId = ref<number | null>(null)
const errorMessage = ref('')
const successMessage = ref('')
const isEditing = computed(() => activePostId.value !== null)

function newPostForm(): PostForm {
  return {
    title: '',
    slug: '',
    summary: '',
    shareImageUrl: '',
    content: '',
    tags: '',
    status: 'draft',
  }
}

function splitTags(value: string): string[] {
  return value.split(',').map(tag => tag.trim()).filter(Boolean)
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
  const response = await fetch(`${apiBase}${path}`, { ...options, credentials: 'include' })
  if (!response.ok) {
    throw new ApiError(response.status, await responseError(response))
  }
  return response
}

async function loadPosts(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''
  try {
    const response = await requestApi('/api/v1/admin/posts')
    posts.value = await response.json() as AdminPost[]
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await navigateTo('/admin/login?reason=session-expired')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '无法加载文章。'
  } finally {
    isLoading.value = false
  }
}

function startCreate(): void {
  activePostId.value = null
  form.value = newPostForm()
  errorMessage.value = ''
  successMessage.value = ''
}

function startEdit(post: AdminPost): void {
  activePostId.value = post.id
  form.value = {
    title: post.title,
    slug: post.slug,
    summary: post.summary,
    shareImageUrl: post.shareImageUrl,
    content: post.content,
    tags: post.tags.join(', '),
    status: post.status,
  }
  errorMessage.value = ''
  successMessage.value = ''
}

async function savePost(): Promise<void> {
  isSaving.value = true
  errorMessage.value = ''
  successMessage.value = ''
  const isCreating = activePostId.value === null
  const path = activePostId.value === null ? '/api/v1/admin/posts' : `/api/v1/admin/posts/${activePostId.value}`

  try {
    await requestApi(path, {
      method: activePostId.value === null ? 'POST' : 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        ...form.value,
        tags: splitTags(form.value.tags),
      }),
    })
    await loadPosts()
    activePostId.value = null
    form.value = newPostForm()
    successMessage.value = isCreating ? '文章已创建。' : '文章已更新。'
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await navigateTo('/admin/login?reason=session-expired')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '保存文章失败。'
  } finally {
    isSaving.value = false
  }
}

async function deletePost(post: AdminPost): Promise<void> {
  if (!window.confirm(`永久删除“${post.title}”？此操作无法恢复。`)) {
    return
  }
  deletingId.value = post.id
  errorMessage.value = ''
  successMessage.value = ''
  try {
    await requestApi(`/api/v1/admin/posts/${post.id}`, { method: 'DELETE' })
    if (activePostId.value === post.id) {
      startCreate()
    }
    successMessage.value = '文章已永久删除。'
    await loadPosts()
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      await navigateTo('/admin/login?reason=session-expired')
      return
    }
    errorMessage.value = error instanceof Error ? error.message : '删除文章失败。'
  } finally {
    deletingId.value = null
  }
}

function statusLabel(status: PostStatus): string {
  if (status === 'published') return '已发布'
  if (status === 'archived') return '已归档'
  return '草稿'
}

onMounted(loadPosts)
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-4 py-10 text-zinc-100 sm:px-6 lg:px-8">
    <section class="mx-auto max-w-7xl">
      <NuxtLink to="/admin" class="text-sm text-zinc-500 transition hover:text-zinc-200">返回管理控制台</NuxtLink>

      <header class="mt-8 flex flex-wrap items-end justify-between gap-5 border-b border-zinc-800 pb-7">
        <div>
          <p class="text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">Administration / Blog</p>
          <h1 class="mt-3 text-3xl font-semibold">博客管理</h1>
        </div>
        <button type="button" class="border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300" @click="startCreate">新建文章</button>
      </header>
      <p class="mt-5 text-sm text-zinc-400">文章图片可先在 <NuxtLink to="/admin/media" class="text-violet-300 underline underline-offset-4">媒体库</NuxtLink> 上传并复制公开地址，再插入 Markdown。</p>

      <p v-if="errorMessage" class="mt-6 border border-rose-400/40 bg-rose-400/10 p-4 text-sm text-rose-100" role="alert">{{ errorMessage }}</p>
      <p v-if="successMessage" class="mt-6 border border-emerald-400/40 bg-emerald-400/10 p-4 text-sm text-emerald-100" role="status">{{ successMessage }}</p>

      <div class="mt-8 grid gap-8 xl:grid-cols-[minmax(0,0.8fr)_minmax(30rem,1.2fr)]">
        <section aria-labelledby="post-list-heading">
          <div class="flex items-baseline justify-between border-b border-zinc-800 pb-3">
            <h2 id="post-list-heading" class="text-lg font-semibold">全部文章</h2>
            <span class="text-sm text-zinc-500">{{ posts.length }} 篇</span>
          </div>
          <p v-if="isLoading" class="py-6 text-sm text-zinc-400" role="status">正在读取文章。</p>
          <p v-else-if="posts.length === 0" class="border-b border-zinc-800 py-8 text-sm text-zinc-400">还没有文章。先创建一份草稿。</p>
          <ol v-else class="divide-y divide-zinc-800">
            <li v-for="post in posts" :key="post.id" class="py-5">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="text-lg font-semibold">{{ post.title }}</h3>
                    <span class="border px-2 py-1 text-xs" :class="post.status === 'published' ? 'border-emerald-400/40 text-emerald-200' : post.status === 'archived' ? 'border-amber-400/40 text-amber-200' : 'border-zinc-700 text-zinc-400'">{{ statusLabel(post.status) }}</span>
                  </div>
                  <p class="mt-2 truncate text-sm text-zinc-500">/blog/{{ post.slug }}</p>
                  <p class="mt-3 line-clamp-2 text-sm leading-6 text-zinc-400">{{ post.summary }}</p>
                  <p v-if="post.publishedAt" class="mt-3 text-xs text-zinc-500">首次发布：{{ new Date(post.publishedAt).toLocaleDateString('zh-CN') }}</p>
                </div>
                <div class="flex shrink-0 gap-2">
                  <button type="button" class="border border-zinc-700 px-3 py-2 text-sm transition hover:border-zinc-300" @click="startEdit(post)">编辑</button>
                  <button type="button" class="border border-rose-400/40 px-3 py-2 text-sm text-rose-200 transition hover:border-rose-200 disabled:opacity-50" :disabled="deletingId === post.id" @click="deletePost(post)">{{ deletingId === post.id ? '删除中' : '删除' }}</button>
                </div>
              </div>
            </li>
          </ol>
        </section>

        <section class="border border-zinc-800 bg-zinc-900/30 p-5 sm:p-6" aria-labelledby="post-form-heading">
          <div class="border-b border-zinc-800 pb-4">
            <p class="text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">{{ isEditing ? 'Edit entry' : 'New entry' }}</p>
            <h2 id="post-form-heading" class="mt-2 text-xl font-semibold">{{ isEditing ? '编辑文章' : '新建文章' }}</h2>
          </div>
          <form class="mt-6 space-y-5" @submit.prevent="savePost">
            <div class="grid gap-5 sm:grid-cols-2">
              <label class="block text-sm font-medium">标题<input v-model.trim="form.title" required maxlength="160" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" /></label>
              <label class="block text-sm font-medium">稳定 slug<input v-model.trim="form.slug" required pattern="[a-z0-9]+(-[a-z0-9]+)*" maxlength="160" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 font-mono outline-none transition focus:border-emerald-300" /></label>
            </div>
            <label class="block text-sm font-medium">摘要<textarea v-model.trim="form.summary" required maxlength="600" rows="3" class="mt-2 w-full resize-y border border-zinc-700 bg-zinc-950 px-3 py-2 leading-6 outline-none transition focus:border-emerald-300" /></label>
            <label class="block text-sm font-medium">分享封面 HTTPS 地址<input v-model.trim="form.shareImageUrl" type="url" inputmode="url" maxlength="2048" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" placeholder="https://cdn.example.com/share.webp" /><span class="mt-2 block text-xs font-normal text-zinc-500">留空时使用站点默认分享封面；可从媒体库复制公开地址。</span></label>
            <label class="block text-sm font-medium">Markdown 内容<textarea v-model="form.content" required maxlength="100000" rows="18" class="mt-2 w-full resize-y border border-zinc-700 bg-zinc-950 px-3 py-2 font-mono text-sm leading-6 outline-none transition focus:border-emerald-300" /></label>
            <div class="grid gap-5 sm:grid-cols-2">
              <label class="block text-sm font-medium">标签<input v-model="form.tags" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" placeholder="Go, Vue" /></label>
              <label class="block text-sm font-medium">状态<select v-model="form.status" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300"><option value="draft">草稿</option><option value="published">已发布</option><option value="archived">已归档</option></select></label>
            </div>
            <div class="flex flex-wrap gap-3 border-t border-zinc-800 pt-5">
              <button type="submit" class="bg-emerald-300 px-4 py-2 text-sm font-semibold text-zinc-950 transition hover:bg-emerald-200 disabled:cursor-wait disabled:opacity-50" :disabled="isSaving">{{ isSaving ? '保存中' : isEditing ? '保存修改' : '创建文章' }}</button>
              <button v-if="isEditing" type="button" class="border border-zinc-700 px-4 py-2 text-sm transition hover:border-zinc-300" :disabled="isSaving" @click="startCreate">取消编辑</button>
            </div>
          </form>
        </section>
      </div>
    </section>
  </main>
</template>
