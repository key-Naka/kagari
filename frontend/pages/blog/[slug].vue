<script setup lang="ts">
import { computed } from 'vue'

interface PublicPostDetail {
  title: string
  slug: string
  summary: string
  shareImageUrl: string
  tags: string[]
  publishedAt: string
  content: string
}

const route = useRoute()
const slug = typeof route.params.slug === 'string' ? route.params.slug : ''
const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')

const { data, status, error, refresh } = await useAsyncData(`public-blog-post-${slug}`, async () => {
  if (!slug) {
    throw createError({ statusCode: 404, statusMessage: '文章不存在。' })
  }
  let result: unknown
  try {
    result = await $fetch<unknown>(`${apiBase}/api/v1/posts/${slug}`)
  } catch (fetchError) {
    if (isNotFoundError(fetchError)) {
      throw createError({ statusCode: 404, statusMessage: '文章不存在。' })
    }
    throw fetchError
  }
  const post = parsePostDetail(result)
  if (!post) {
    throw new Error('文章公开数据格式无效。')
  }
  return post
}, {
  default: () => null,
})

const isLoading = computed(() => status.value === 'pending')
const errorMessage = computed(() => (
  error.value instanceof Error ? error.value.message : '文章暂时无法获取。'
))

usePublicSeo({
  title: () => data.value ? `${data.value.title} · 博客 · Kagari` : '博客文章 · Kagari',
  description: () => data.value?.summary ?? '阅读 Kagari 的博客文章与公开技术记录。',
  image: () => data.value?.shareImageUrl,
  type: 'article',
})

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function isNotFoundError(value: unknown): boolean {
  return isRecord(value) && (value.status === 404 || value.statusCode === 404)
}

function parseStringList(value: unknown): string[] | null {
  return Array.isArray(value) && value.every(item => typeof item === 'string') ? value : null
}

function parsePostDetail(value: unknown): PublicPostDetail | null {
  if (!isRecord(value)) {
    return null
  }
  const tags = parseStringList(value.tags)
  if (
    typeof value.title !== 'string'
    || typeof value.slug !== 'string'
    || typeof value.summary !== 'string'
    || typeof value.shareImageUrl !== 'string'
    || tags === null
    || typeof value.publishedAt !== 'string'
    || typeof value.content !== 'string'
  ) {
    return null
  }
  return {
    title: value.title,
    slug: value.slug,
    summary: value.summary,
    shareImageUrl: value.shareImageUrl,
    tags,
    publishedAt: value.publishedAt,
    content: value.content,
  }
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  }).format(new Date(value))
}
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-6 py-12 text-zinc-100 sm:px-10 sm:py-16">
    <article class="mx-auto max-w-3xl">
      <NuxtLink to="/blog" class="text-sm text-zinc-500 transition hover:text-zinc-200">返回博客目录</NuxtLink>

      <p v-if="isLoading && !data" class="mt-10 text-zinc-400" role="status" aria-live="polite">正在读取文章。</p>
      <div v-else-if="error || !data" class="mt-10 border border-rose-400/40 bg-rose-400/10 p-5 text-rose-100" role="alert">
        <p>{{ errorMessage }}</p>
        <button type="button" class="mt-4 border border-rose-200/50 px-3 py-2 text-sm transition hover:border-rose-100" @click="refresh()">重试</button>
      </div>

      <template v-else>
        <header class="mt-8 border-b border-zinc-800 pb-8">
          <time :datetime="data.publishedAt" class="text-xs font-medium uppercase tracking-[0.2em] text-zinc-500">{{ formatDate(data.publishedAt) }}</time>
          <h1 class="mt-4 text-4xl font-semibold sm:text-5xl">{{ data.title }}</h1>
          <p class="mt-5 text-lg leading-8 text-zinc-400">{{ data.summary }}</p>
          <div v-if="data.tags.length > 0" class="mt-6 flex flex-wrap gap-2">
            <NuxtLink v-for="tag in data.tags" :key="tag" :to="{ path: '/blog', query: { tag } }" class="border border-zinc-700 px-2 py-1 text-xs text-zinc-300 transition hover:border-emerald-300 hover:text-emerald-200">{{ tag }}</NuxtLink>
          </div>
        </header>

        <div class="mt-10 leading-8 text-zinc-300 [&_a]:text-emerald-200 [&_a]:underline [&_blockquote]:border-l-2 [&_blockquote]:border-zinc-600 [&_blockquote]:pl-5 [&_blockquote]:text-zinc-400 [&_code]:bg-zinc-900 [&_code]:px-1.5 [&_code]:py-0.5 [&_figcaption]:mt-3 [&_figcaption]:text-sm [&_figcaption]:text-zinc-500 [&_figure]:my-8 [&_h1]:mt-10 [&_h1]:text-3xl [&_h1]:font-semibold [&_h2]:mt-10 [&_h2]:text-2xl [&_h2]:font-semibold [&_h3]:mt-8 [&_h3]:text-xl [&_h3]:font-semibold [&_img]:w-full [&_li]:my-2 [&_ol]:my-6 [&_ol]:list-decimal [&_ol]:pl-6 [&_p]:my-6 [&_pre]:my-6 [&_pre]:overflow-x-auto [&_pre]:border [&_pre]:border-zinc-800 [&_pre]:bg-zinc-900 [&_pre]:p-4 [&_ul]:my-6 [&_ul]:list-disc [&_ul]:pl-6" v-html="data.content" />
      </template>
    </article>
  </main>
</template>
