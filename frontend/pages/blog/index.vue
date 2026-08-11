<script setup lang="ts">
import { computed } from 'vue'

interface PublicPost {
  title: string
  slug: string
  summary: string
  tags: string[]
  publishedAt: string
}

interface PostTag {
  name: string
  count: number
}

interface Archive {
  key: string
  year: number
  month: number
  count: number
}

interface NavigationData {
  tags: PostTag[]
  archives: Archive[]
}

const route = useRoute()
const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const selectedTag = computed(() => typeof route.query.tag === 'string' ? route.query.tag : '')
const selectedArchive = computed(() => typeof route.query.archive === 'string' ? route.query.archive : '')

const { data, status, error, refresh } = await useAsyncData('public-blog-posts', async () => {
  const query = new URLSearchParams()
  if (selectedTag.value) query.set('tag', selectedTag.value)
  if (selectedArchive.value) query.set('archive', selectedArchive.value)
  const suffix = query.size > 0 ? `?${query.toString()}` : ''
  const result = await $fetch<unknown>(`${apiBase}/api/v1/posts${suffix}`)
  const posts = parsePostList(result)
  if (!posts) {
    throw new Error('博客公开数据格式无效。')
  }
  return posts
}, {
  default: () => [],
  watch: [selectedTag, selectedArchive],
})

const { data: navigation, error: navigationError } = await useAsyncData('public-blog-navigation', async () => {
  const [tagsResult, archivesResult] = await Promise.all([
    $fetch<unknown>(`${apiBase}/api/v1/posts/tags`),
    $fetch<unknown>(`${apiBase}/api/v1/posts/archives`),
  ])
  const tags = parseTagList(tagsResult)
  const archives = parseArchiveList(archivesResult)
  if (!tags || !archives) {
    throw new Error('博客筛选数据格式无效。')
  }
  return { tags, archives }
}, {
  default: (): NavigationData => ({ tags: [], archives: [] }),
})

const isLoading = computed(() => status.value === 'pending')
const errorMessage = computed(() => (
  error.value instanceof Error ? error.value.message : '博客目录暂时无法获取。'
))
const navigationErrorMessage = computed(() => (
  navigationError.value instanceof Error ? navigationError.value.message : '筛选目录暂时无法获取。'
))

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function parseStringList(value: unknown): string[] | null {
  return Array.isArray(value) && value.every(item => typeof item === 'string') ? value : null
}

function parsePost(value: unknown): PublicPost | null {
  if (!isRecord(value)) {
    return null
  }
  const tags = parseStringList(value.tags)
  if (
    typeof value.title !== 'string'
    || typeof value.slug !== 'string'
    || typeof value.summary !== 'string'
    || tags === null
    || typeof value.publishedAt !== 'string'
  ) {
    return null
  }
  return {
    title: value.title,
    slug: value.slug,
    summary: value.summary,
    tags,
    publishedAt: value.publishedAt,
  }
}

function parsePostList(value: unknown): PublicPost[] | null {
  if (!Array.isArray(value)) {
    return null
  }
  const posts = value.map(parsePost)
  return posts.every((post): post is PublicPost => post !== null) ? posts : null
}

function parseTag(value: unknown): PostTag | null {
  if (!isRecord(value) || typeof value.name !== 'string' || typeof value.count !== 'number') {
    return null
  }
  return { name: value.name, count: value.count }
}

function parseTagList(value: unknown): PostTag[] | null {
  if (!Array.isArray(value)) {
    return null
  }
  const tags = value.map(parseTag)
  return tags.every((tag): tag is PostTag => tag !== null) ? tags : null
}

function parseArchive(value: unknown): Archive | null {
  if (
    !isRecord(value)
    || typeof value.key !== 'string'
    || typeof value.year !== 'number'
    || typeof value.month !== 'number'
    || typeof value.count !== 'number'
  ) {
    return null
  }
  return { key: value.key, year: value.year, month: value.month, count: value.count }
}

function parseArchiveList(value: unknown): Archive[] | null {
  if (!Array.isArray(value)) {
    return null
  }
  const archives = value.map(parseArchive)
  return archives.every((archive): archive is Archive => archive !== null) ? archives : null
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  }).format(new Date(value))
}

async function setFilters(tag = '', archive = ''): Promise<void> {
  const query: Record<string, string> = {}
  if (tag) query.tag = tag
  if (archive) query.archive = archive
  await navigateTo({ path: '/blog', query })
}
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-6 py-12 text-zinc-100 sm:px-10 sm:py-16">
    <div class="mx-auto max-w-6xl">
      <NuxtLink to="/" class="text-sm text-zinc-500 transition hover:text-zinc-200">返回首页</NuxtLink>

      <header class="mt-8 flex flex-wrap items-end justify-between gap-6 border-b border-zinc-800 pb-8">
        <div>
          <p class="text-xs font-medium uppercase tracking-[0.28em] text-zinc-500">Archive / Writing</p>
          <h1 class="mt-3 text-4xl font-semibold sm:text-5xl">博客</h1>
          <p class="mt-3 max-w-2xl leading-7 text-zinc-400">按主题与时间整理的公开文章。</p>
        </div>
        <button
          type="button"
          class="border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300 disabled:cursor-wait disabled:opacity-60"
          :disabled="isLoading"
          @click="refresh()"
        >
          {{ isLoading ? '正在读取' : '刷新' }}
        </button>
      </header>

      <div class="mt-8 grid gap-10 lg:grid-cols-[minmax(0,1fr)_14rem]">
        <section aria-label="已发布文章">
          <p v-if="isLoading && data.length === 0" class="text-zinc-400" role="status" aria-live="polite">正在读取公开文章。</p>
          <div v-else-if="error" class="border border-rose-400/40 bg-rose-400/10 p-5 text-rose-100" role="alert">
            <p>{{ errorMessage }}</p>
            <button type="button" class="mt-4 border border-rose-200/50 px-3 py-2 text-sm transition hover:border-rose-100" @click="refresh()">重试</button>
          </div>
          <p v-else-if="data.length === 0" class="border-y border-zinc-800 py-8 text-zinc-400">没有符合当前筛选条件的已发布文章。</p>
          <ol v-else class="divide-y divide-zinc-800 border-y border-zinc-800">
            <li v-for="post in data" :key="post.slug">
              <NuxtLink :to="`/blog/${post.slug}`" class="block py-7 transition hover:bg-zinc-900/40 sm:px-4">
                <time :datetime="post.publishedAt" class="text-xs font-medium uppercase tracking-[0.18em] text-zinc-500">{{ formatDate(post.publishedAt) }}</time>
                <h2 class="mt-3 text-2xl font-semibold">{{ post.title }}</h2>
                <p class="mt-3 max-w-3xl leading-7 text-zinc-400">{{ post.summary }}</p>
                <div v-if="post.tags.length > 0" class="mt-5 flex flex-wrap gap-2">
                  <span v-for="tag in post.tags" :key="tag" class="border border-zinc-700 px-2 py-1 text-xs text-zinc-300">{{ tag }}</span>
                </div>
              </NuxtLink>
            </li>
          </ol>
        </section>

        <aside class="border-t border-zinc-800 pt-6 lg:border-l lg:border-t-0 lg:pl-6 lg:pt-0" aria-label="博客筛选">
          <p v-if="navigationError" class="text-sm leading-6 text-rose-200" role="alert">{{ navigationErrorMessage }}</p>
          <template v-else>
            <section>
              <h2 class="text-sm font-medium uppercase tracking-[0.2em] text-zinc-500">标签</h2>
              <div class="mt-4 flex flex-wrap gap-2 lg:flex-col lg:items-stretch">
                <button type="button" class="border px-3 py-2 text-left text-sm transition" :class="!selectedTag && !selectedArchive ? 'border-emerald-300 text-emerald-200' : 'border-zinc-800 text-zinc-400 hover:border-zinc-500'" @click="setFilters()">全部文章</button>
                <button v-for="tag in navigation.tags" :key="tag.name" type="button" class="flex border px-3 py-2 text-left text-sm transition" :class="selectedTag === tag.name ? 'border-emerald-300 text-emerald-200' : 'border-zinc-800 text-zinc-400 hover:border-zinc-500'" @click="setFilters(tag.name)">
                  <span>{{ tag.name }}</span><span class="ml-auto text-zinc-500">{{ tag.count }}</span>
                </button>
              </div>
            </section>

            <section class="mt-8">
              <h2 class="text-sm font-medium uppercase tracking-[0.2em] text-zinc-500">年月归档</h2>
              <div class="mt-4 flex flex-wrap gap-2 lg:flex-col lg:items-stretch">
                <button v-for="archive in navigation.archives" :key="archive.key" type="button" class="flex border px-3 py-2 text-left text-sm transition" :class="selectedArchive === archive.key ? 'border-emerald-300 text-emerald-200' : 'border-zinc-800 text-zinc-400 hover:border-zinc-500'" @click="setFilters('', archive.key)">
                  <span>{{ archive.year }} 年 {{ archive.month }} 月</span><span class="ml-auto text-zinc-500">{{ archive.count }}</span>
                </button>
              </div>
            </section>
          </template>
        </aside>
      </div>
    </div>
  </main>
</template>
