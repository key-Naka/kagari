<script setup lang="ts">
import { computed } from 'vue'

defineOptions({
  name: 'PublicGitHubPage',
})

type Availability = 'operational' | 'degraded' | 'unavailable'
type PageState = 'idle' | 'loading' | 'ready' | 'error'

interface ContributionDay {
  date: string
  level: number
}

interface GitHubActivity {
  kind: string
  repository: string
  occurredAt: string
}

interface GitHubRepository {
  name: string
  url: string
  description: string
  language: string
  stars: number
  updatedAt: string
}

interface GitHubActivityData {
  availability: Availability
  contributions: ContributionDay[]
  activities: GitHubActivity[]
  repositories: GitHubRepository[]
  sampledAt: string
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const { data, status, error, refresh } = await useAsyncData('public-github-activity', async () => {
  const result = await $fetch<unknown>(`${apiBase}/api/v1/github`)
  const parsed = parseGitHubActivityData(result)
  if (!parsed) {
    throw new Error('GitHub 公开数据格式无效。')
  }
  return parsed
}, {
  default: () => null,
})

const pageState = computed<PageState>(() => {
  if (status.value === 'pending') {
    return 'loading'
  }
  if (status.value === 'error') {
    return 'error'
  }
  return data.value ? 'ready' : 'idle'
})

const errorMessage = computed(() => (
  error.value instanceof Error ? error.value.message : 'GitHub 公开数据暂时无法获取。'
))

const availabilityLabel = computed(() => {
  if (data.value?.availability === 'operational') {
    return '同步正常'
  }
  return '使用最近快照'
})

const availabilityClass = computed(() => (
  data.value?.availability === 'operational'
    ? 'border-emerald-400/40 bg-emerald-400/10 text-emerald-100'
    : 'border-amber-400/40 bg-amber-400/10 text-amber-100'
))

const sampledAt = computed(() => {
  const value = data.value?.sampledAt
  if (!value) {
    return ''
  }
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
})

function isAvailability(value: unknown): value is Availability {
  return value === 'operational' || value === 'degraded' || value === 'unavailable'
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function parseContribution(value: unknown): ContributionDay | null {
  if (!isRecord(value) || typeof value.date !== 'string' || typeof value.level !== 'number' || value.level < 0 || value.level > 4) {
    return null
  }
  return { date: value.date, level: value.level }
}

function parseActivity(value: unknown): GitHubActivity | null {
  if (!isRecord(value) || typeof value.kind !== 'string' || typeof value.repository !== 'string' || typeof value.occurredAt !== 'string') {
    return null
  }
  return { kind: value.kind, repository: value.repository, occurredAt: value.occurredAt }
}

function parseRepository(value: unknown): GitHubRepository | null {
  if (
    !isRecord(value)
    || typeof value.name !== 'string'
    || typeof value.url !== 'string'
    || typeof value.description !== 'string'
    || typeof value.language !== 'string'
    || typeof value.stars !== 'number'
    || typeof value.updatedAt !== 'string'
  ) {
    return null
  }
  return {
    name: value.name,
    url: value.url,
    description: value.description,
    language: value.language,
    stars: value.stars,
    updatedAt: value.updatedAt,
  }
}

function parseGitHubActivityData(value: unknown): GitHubActivityData | null {
  if (!isRecord(value) || !isAvailability(value.availability) || !Array.isArray(value.contributions) || !Array.isArray(value.activities) || !Array.isArray(value.repositories) || typeof value.sampledAt !== 'string') {
    return null
  }

  const contributions = value.contributions.map(parseContribution)
  const activities = value.activities.map(parseActivity)
  const repositories = value.repositories.map(parseRepository)

  if (contributions.some((item) => item === null) || activities.some((item) => item === null) || repositories.some((item) => item === null)) {
    return null
  }

  return {
    availability: value.availability,
    contributions: contributions as ContributionDay[],
    activities: activities as GitHubActivity[],
    repositories: repositories as GitHubRepository[],
    sampledAt: value.sampledAt,
  }
}

function heatmapClass(level: number): string {
  const classes = [
    'bg-zinc-900',
    'bg-emerald-950',
    'bg-emerald-800',
    'bg-emerald-600',
    'bg-emerald-400',
  ]
  return classes[level] ?? 'bg-zinc-900'
}

function formatDate(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function formatDateTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
}

async function refreshGitHubActivity(): Promise<void> {
  await refresh()
}

usePublicSeo({
  title: 'GitHub 档案 · Kagari',
  description: '查看 Kagari 的公开贡献热力、近期工程活动与精选 GitHub 仓库。',
})
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-6 py-12 text-zinc-100 sm:px-10 sm:py-16">
    <div class="mx-auto max-w-6xl">
      <NuxtLink class="text-sm text-zinc-500 transition hover:text-zinc-200" to="/">返回首页</NuxtLink>

      <header class="mt-8 flex flex-wrap items-end justify-between gap-6 border-b border-zinc-800 pb-8">
        <div>
          <p class="text-xs font-medium uppercase tracking-[0.28em] text-zinc-500">Public Signal / key-Naka</p>
          <h1 class="mt-3 text-4xl font-semibold sm:text-5xl">GitHub 活动</h1>
          <p class="mt-3 max-w-2xl leading-7 text-zinc-400">公开贡献、近期动态与最近更新的仓库。数据每小时缓存，并在上游暂不可用时保留最近一次成功快照。</p>
        </div>
        <div class="flex items-center gap-3">
          <span v-if="data" class="rounded-full border px-3 py-1.5 text-sm" :class="availabilityClass">{{ availabilityLabel }}</span>
          <button
            class="border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300 disabled:cursor-wait disabled:opacity-60"
            type="button"
            :disabled="pageState === 'loading'"
            @click="refreshGitHubActivity"
          >
            {{ pageState === 'loading' ? '正在刷新' : '刷新' }}
          </button>
        </div>
      </header>

      <p v-if="pageState === 'loading' && !data" class="mt-8 text-zinc-400" role="status" aria-live="polite">正在读取公开 GitHub 数据。</p>
      <p v-else-if="pageState === 'error'" class="mt-8 border border-rose-400/40 bg-rose-400/10 p-4 text-rose-100" role="alert">{{ errorMessage }}</p>
      <p v-else-if="data && data.availability !== 'operational'" class="mt-8 border border-amber-400/40 bg-amber-400/10 p-4 text-amber-100" role="alert">GitHub 上游暂不可用，以下内容来自最近一次成功快照。</p>

      <template v-if="data">
        <section class="mt-10 border border-zinc-800 bg-zinc-900/40 p-5 sm:p-7" aria-labelledby="contribution-heading">
          <div class="flex flex-wrap items-end justify-between gap-3">
            <div>
              <p class="text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">Last 12 months</p>
              <h2 id="contribution-heading" class="mt-2 text-2xl font-semibold">贡献热力图</h2>
            </div>
            <p class="text-sm text-zinc-500">{{ data.contributions.length }} 个采样日</p>
          </div>
          <div class="mt-7 overflow-x-auto pb-2">
            <div class="grid min-w-[700px] grid-flow-col grid-rows-7 gap-1" aria-label="过去十二个月的 GitHub 贡献热力图">
              <span
                v-for="contribution in data.contributions"
                :key="contribution.date"
                class="h-3 w-3"
                :class="heatmapClass(contribution.level)"
                :title="`${contribution.date}：等级 ${contribution.level}`"
              />
            </div>
          </div>
          <div class="mt-4 flex items-center justify-end gap-1 text-xs text-zinc-500">
            <span>少</span>
            <span v-for="level in [0, 1, 2, 3, 4]" :key="level" class="h-3 w-3" :class="heatmapClass(level)" />
            <span>多</span>
          </div>
        </section>

        <div class="mt-10 grid gap-10 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.15fr)]">
          <section aria-labelledby="activity-heading">
            <div class="flex items-end justify-between gap-4 border-b border-zinc-800 pb-4">
              <div>
                <p class="text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">Recent activity</p>
                <h2 id="activity-heading" class="mt-2 text-2xl font-semibold">近期动态</h2>
              </div>
            </div>
            <ul v-if="data.activities.length" class="divide-y divide-zinc-800">
              <li v-for="activity in data.activities" :key="`${activity.repository}-${activity.occurredAt}-${activity.kind}`" class="py-5">
                <p class="font-medium text-zinc-100">{{ activity.kind }} · {{ activity.repository }}</p>
                <p class="mt-2 text-sm text-zinc-500">{{ formatDateTime(activity.occurredAt) }}</p>
              </li>
            </ul>
            <p v-else class="py-8 text-sm text-zinc-500">暂时没有可展示的公开动态。</p>
          </section>

          <section aria-labelledby="repository-heading">
            <div class="flex items-end justify-between gap-4 border-b border-zinc-800 pb-4">
              <div>
                <p class="text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">Recently updated</p>
                <h2 id="repository-heading" class="mt-2 text-2xl font-semibold">公开仓库</h2>
              </div>
            </div>
            <div v-if="data.repositories.length" class="mt-5 grid gap-4 sm:grid-cols-2">
              <article v-for="repository in data.repositories" :key="repository.url" class="border border-zinc-800 bg-zinc-900/40 p-5">
                <div class="flex items-start justify-between gap-4">
                  <h3 class="text-lg font-semibold">{{ repository.name }}</h3>
                  <span v-if="repository.stars" class="shrink-0 text-sm text-amber-200">{{ repository.stars }} stars</span>
                </div>
                <p class="mt-3 min-h-12 text-sm leading-6 text-zinc-400">{{ repository.description || '暂无公开说明。' }}</p>
                <div class="mt-5 flex items-center justify-between gap-3 text-sm">
                  <span class="text-zinc-500">{{ repository.language || '未标注语言' }}</span>
                  <a class="text-emerald-200 transition hover:text-emerald-100" :href="repository.url" target="_blank" rel="noopener noreferrer">查看仓库</a>
                </div>
                <p class="mt-4 text-xs text-zinc-600">更新于 {{ formatDate(repository.updatedAt) }}</p>
              </article>
            </div>
            <p v-else class="py-8 text-sm text-zinc-500">暂时没有可展示的公开仓库。</p>
          </section>
        </div>

        <p class="mt-12 border-t border-zinc-800 pt-5 text-sm text-zinc-500">最近采样：{{ sampledAt }}</p>
      </template>
    </div>
  </main>
</template>
