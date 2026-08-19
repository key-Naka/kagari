<script setup lang="ts">
import { computed } from 'vue'

interface PublicProject {
  title: string
  slug: string
  coverUrl: string
  description: string
  technologies: string[]
  types: string[]
  featured: boolean
  sortOrder: number
  websiteUrl: string
  repositoryUrl: string
}

const route = useRoute()
const slug = typeof route.params.slug === 'string' ? route.params.slug : ''
const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const { data, status, error, refresh } = await useAsyncData(`public-project-${slug}`, async () => {
  if (!slug) {
    throw createError({ statusCode: 404, statusMessage: '作品不存在。' })
  }
  const result = await $fetch<unknown>(`${apiBase}/api/v1/projects/${slug}`)
  const project = parseProject(result)
  if (!project) {
    throw new Error('作品公开数据格式无效。')
  }
  return project
}, {
  default: () => null,
})

const isLoading = computed(() => status.value === 'pending')
const errorMessage = computed(() => (
  error.value instanceof Error ? error.value.message : '作品暂时无法获取。'
))

usePublicSeo({
  title: () => data.value ? `${data.value.title} · 作品 · Kagari` : '作品详情 · Kagari',
  description: () => data.value?.description ?? '阅读 Kagari 的作品详情、技术选择与项目入口。',
  image: () => data.value?.coverUrl,
  type: 'article',
})

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function parseStringList(value: unknown): string[] | null {
  return Array.isArray(value) && value.every(item => typeof item === 'string') ? value : null
}

function parseProject(value: unknown): PublicProject | null {
  if (!isRecord(value)) {
    return null
  }
  const technologies = parseStringList(value.technologies)
  const types = parseStringList(value.types)
  if (
    typeof value.title !== 'string'
    || typeof value.slug !== 'string'
    || typeof value.coverUrl !== 'string'
    || typeof value.description !== 'string'
    || technologies === null
    || types === null
    || typeof value.featured !== 'boolean'
    || typeof value.sortOrder !== 'number'
    || typeof value.websiteUrl !== 'string'
    || typeof value.repositoryUrl !== 'string'
  ) {
    return null
  }
  return {
    title: value.title,
    slug: value.slug,
    coverUrl: value.coverUrl,
    description: value.description,
    technologies,
    types,
    featured: value.featured,
    sortOrder: value.sortOrder,
    websiteUrl: value.websiteUrl,
    repositoryUrl: value.repositoryUrl,
  }
}
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-6 py-12 text-zinc-100 sm:px-10 sm:py-16">
    <article class="mx-auto max-w-5xl">
      <NuxtLink to="/works" class="text-sm text-zinc-500 transition hover:text-zinc-200">返回作品目录</NuxtLink>

      <p v-if="isLoading && !data" class="mt-10 text-zinc-400" role="status" aria-live="polite">正在读取作品档案。</p>
      <div v-else-if="error || !data" class="mt-10 border border-rose-400/40 bg-rose-400/10 p-5 text-rose-100" role="alert">
        <p>{{ errorMessage }}</p>
        <button type="button" class="mt-4 border border-rose-200/50 px-3 py-2 text-sm transition hover:border-rose-100" @click="refresh()">重试</button>
      </div>

      <template v-else>
        <header class="mt-8 border-b border-zinc-800 pb-8">
          <div class="flex flex-wrap items-center gap-3 text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">
            <span>Archive / Work</span>
            <span v-if="data.featured" class="border border-emerald-400/40 px-2 py-1 text-emerald-200">精选</span>
          </div>
          <h1 class="mt-5 max-w-4xl text-4xl font-semibold sm:text-6xl">{{ data.title }}</h1>
          <p class="mt-6 max-w-3xl text-lg leading-8 text-zinc-400">{{ data.description }}</p>
        </header>

        <img :src="data.coverUrl" :alt="`${data.title} 封面`" class="mt-10 aspect-[16/9] w-full border border-zinc-800 object-cover">

        <section class="mt-10 grid gap-8 border-t border-zinc-800 pt-8 md:grid-cols-[0.7fr_1.3fr]">
          <h2 class="text-sm font-medium uppercase tracking-[0.24em] text-zinc-500">技术与类型</h2>
          <div class="space-y-6">
            <div>
              <p class="text-sm text-zinc-500">技术</p>
              <div class="mt-3 flex flex-wrap gap-2">
                <span v-for="technology in data.technologies" :key="technology" class="border border-zinc-700 px-2 py-1 text-sm text-zinc-200">{{ technology }}</span>
              </div>
            </div>
            <div>
              <p class="text-sm text-zinc-500">类型</p>
              <div class="mt-3 flex flex-wrap gap-2">
                <span v-for="type in data.types" :key="type" class="border border-zinc-700 px-2 py-1 text-sm text-zinc-200">{{ type }}</span>
              </div>
            </div>
            <div class="flex flex-wrap gap-3 pt-2">
              <a v-if="data.websiteUrl" :href="data.websiteUrl" target="_blank" rel="noopener noreferrer" class="border border-emerald-300 bg-emerald-300 px-4 py-2 text-sm font-medium text-zinc-950 transition hover:bg-emerald-200">访问网站</a>
              <a v-if="data.repositoryUrl" :href="data.repositoryUrl" target="_blank" rel="noopener noreferrer" class="border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300">查看源码</a>
            </div>
          </div>
        </section>
      </template>
    </article>
  </main>
</template>
