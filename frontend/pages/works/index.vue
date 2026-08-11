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

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const { data, status, error, refresh } = await useAsyncData('public-projects', async () => {
  const result = await $fetch<unknown>(`${apiBase}/api/v1/projects`)
  const projects = parseProjectList(result)
  if (!projects) {
    throw new Error('作品公开数据格式无效。')
  }
  return projects
}, {
  default: () => [],
})

const isLoading = computed(() => status.value === 'pending')
const errorMessage = computed(() => (
  error.value instanceof Error ? error.value.message : '作品目录暂时无法获取。'
))

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

function parseProjectList(value: unknown): PublicProject[] | null {
  if (!Array.isArray(value)) {
    return null
  }
  const projects = value.map(parseProject)
  return projects.every((project): project is PublicProject => project !== null) ? projects : null
}
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-6 py-12 text-zinc-100 sm:px-10 sm:py-16">
    <div class="mx-auto max-w-6xl">
      <NuxtLink to="/" class="text-sm text-zinc-500 transition hover:text-zinc-200">返回首页</NuxtLink>

      <header class="mt-8 flex flex-wrap items-end justify-between gap-6 border-b border-zinc-800 pb-8">
        <div>
          <p class="text-xs font-medium uppercase tracking-[0.28em] text-zinc-500">Archive / 01</p>
          <h1 class="mt-3 text-4xl font-semibold sm:text-5xl">作品</h1>
          <p class="mt-3 max-w-2xl leading-7 text-zinc-400">已发布的工程与界面作品，按精选与排序次序归档。</p>
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

      <p v-if="isLoading && data.length === 0" class="mt-8 text-zinc-400" role="status" aria-live="polite">正在读取公开作品目录。</p>
      <p v-else-if="error" class="mt-8 border border-rose-400/40 bg-rose-400/10 p-4 text-rose-100" role="alert">{{ errorMessage }}</p>
      <p v-else-if="data.length === 0" class="mt-8 border border-zinc-800 bg-zinc-900/40 p-5 text-zinc-400">暂时没有已发布作品。</p>

      <section v-else class="mt-10 grid gap-px overflow-hidden border border-zinc-800 bg-zinc-800 sm:grid-cols-2" aria-label="已发布作品">
        <article v-for="project in data" :key="project.slug" class="group bg-zinc-950">
          <NuxtLink :to="`/works/${project.slug}`" class="block h-full transition hover:bg-zinc-900/50">
            <div class="aspect-[16/10] overflow-hidden bg-zinc-900">
              <img
                :src="project.coverUrl"
                :alt="`${project.title} 封面`"
                class="h-full w-full object-cover transition duration-500 group-hover:scale-[1.03]"
                loading="lazy"
              >
            </div>
            <div class="p-5 sm:p-7">
              <div class="flex items-start justify-between gap-4">
                <h2 class="text-2xl font-semibold">{{ project.title }}</h2>
                <span v-if="project.featured" class="border border-emerald-400/40 px-2 py-1 text-xs text-emerald-200">精选</span>
              </div>
              <p class="mt-4 line-clamp-3 leading-7 text-zinc-400">{{ project.description }}</p>
              <div class="mt-6 flex flex-wrap gap-2">
                <span v-for="technology in project.technologies" :key="technology" class="border border-zinc-700 px-2 py-1 text-xs text-zinc-300">{{ technology }}</span>
              </div>
            </div>
          </NuxtLink>
        </article>
      </section>
    </div>
  </main>
</template>
