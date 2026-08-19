<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

defineOptions({
  name: 'PublicServiceStatusPage',
})

definePageMeta({
  path: '/status',
})

type AvailabilityState = 'operational' | 'degraded' | 'unavailable'
type PageState = 'idle' | 'loading' | 'ready' | 'error'

type ResourceMetric = Readonly<{
  label: string
  value: string
}>

type ContainerStatus = Readonly<{
  name: 'Web' | 'API' | 'Database' | 'Cache'
  state: AvailabilityState
  resources: AvailabilityState
}>

type ApplicationCheck = Readonly<{
  name: 'API' | 'HTTP' | 'MySQL' | 'Redis'
  state: AvailabilityState
}>

interface ServiceStatusDto {
  availability: AvailabilityState
  resources: Readonly<{
    cpu: string
    memory: string
    disk: string
    network: string
    uptime: string
  }>
  containers: readonly ContainerStatus[]
  applications: readonly ApplicationCheck[]
  sampledAt: string
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const requestState = ref<PageState>('idle')
const serviceStatus = ref<ServiceStatusDto | null>(null)
const errorMessage = ref('')

const resourceMetrics = computed<readonly ResourceMetric[]>(() => {
  if (!serviceStatus.value) {
    return []
  }

  const { cpu, memory, disk, network, uptime } = serviceStatus.value.resources
  return [
    { label: 'CPU', value: cpu },
    { label: '内存', value: memory },
    { label: '磁盘', value: disk },
    { label: '网络', value: network },
    { label: '运行时间', value: uptime },
  ]
})

const isDegraded = computed(() => serviceStatus.value?.availability === 'degraded')
const sampledAt = computed(() => {
  const value = serviceStatus.value?.sampledAt
  if (!value) {
    return '尚未采样'
  }

  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
})

function stateLabel(state: AvailabilityState): string {
  const labels: Record<AvailabilityState, string> = {
    operational: '正常',
    degraded: '降级',
    unavailable: '不可用',
  }

  return labels[state]
}

function stateClass(state: AvailabilityState): string {
  const classes: Record<AvailabilityState, string> = {
    operational: 'border-emerald-400/40 bg-emerald-400/10 text-emerald-200',
    degraded: 'border-amber-400/40 bg-amber-400/10 text-amber-100',
    unavailable: 'border-rose-400/40 bg-rose-400/10 text-rose-100',
  }

  return classes[state]
}

function isAvailabilityState(value: unknown): value is AvailabilityState {
  return value === 'operational' || value === 'degraded' || value === 'unavailable'
}

function isStringRecord(value: unknown): value is Record<string, string> {
  return typeof value === 'object' && value !== null && Object.values(value).every((entry) => typeof entry === 'string')
}

function parseServiceStatus(value: unknown): ServiceStatusDto | null {
  if (typeof value !== 'object' || value === null) {
    return null
  }

  const response = value as Record<string, unknown>
  const resources = response.resources
  const containers = response.containers
  const applications = response.applications

  if (
    !isAvailabilityState(response.availability)
    || !isStringRecord(resources)
    || typeof resources.cpu !== 'string'
    || typeof resources.memory !== 'string'
    || typeof resources.disk !== 'string'
    || typeof resources.network !== 'string'
    || typeof resources.uptime !== 'string'
    || !Array.isArray(containers)
    || !Array.isArray(applications)
    || typeof response.sampledAt !== 'string'
  ) {
    return null
  }

  const expectedContainers = ['Web', 'API', 'Database', 'Cache'] as const
  const expectedApplications = ['API', 'HTTP', 'MySQL', 'Redis'] as const
  const parsedContainers: ContainerStatus[] = []
  const parsedApplications: ApplicationCheck[] = []

  for (const name of expectedContainers) {
    const item = containers.find((candidate): candidate is Record<string, unknown> => (
      typeof candidate === 'object' && candidate !== null && (candidate as Record<string, unknown>).name === name
    ))
    if (!item || !isAvailabilityState(item.state) || !isAvailabilityState(item.resources)) {
      return null
    }
    parsedContainers.push({ name, state: item.state, resources: item.resources })
  }

  for (const name of expectedApplications) {
    const item = applications.find((candidate): candidate is Record<string, unknown> => (
      typeof candidate === 'object' && candidate !== null && (candidate as Record<string, unknown>).name === name
    ))
    if (!item || !isAvailabilityState(item.state)) {
      return null
    }
    parsedApplications.push({ name, state: item.state })
  }

  return {
    availability: response.availability,
    resources: {
      cpu: resources.cpu,
      memory: resources.memory,
      disk: resources.disk,
      network: resources.network,
      uptime: resources.uptime,
    },
    containers: parsedContainers,
    applications: parsedApplications,
    sampledAt: response.sampledAt,
  }
}

async function refreshStatus(): Promise<void> {
  requestState.value = 'loading'
  errorMessage.value = ''

  try {
    const response = await fetch(`${apiBase}/api/v1/service-status`)
    if (!response.ok) {
      throw new Error('服务状态暂时无法获取。')
    }

    const result: unknown = await response.json()
    const parsedStatus = parseServiceStatus(result)
    if (!parsedStatus) {
      throw new Error('服务状态响应格式无效。')
    }

    serviceStatus.value = parsedStatus
    requestState.value = 'ready'
  } catch (error: unknown) {
    serviceStatus.value = null
    requestState.value = 'error'
    errorMessage.value = error instanceof Error ? error.message : '服务状态暂时无法获取。'
  }
}

onMounted(() => {
  void refreshStatus()
})

usePublicSeo({
  title: '服务状态 · Kagari',
  description: '查看 Kagari 公开服务的应用状态与经过脱敏处理的基础设施指标。',
})
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-6 py-16 text-zinc-100 sm:px-10">
    <div class="mx-auto max-w-5xl">
      <NuxtLink class="text-sm text-zinc-400 transition hover:text-zinc-100" to="/">返回首页</NuxtLink>
      <header class="mt-8 flex flex-wrap items-end justify-between gap-6 border-b border-zinc-800 pb-8">
        <div>
          <p class="text-sm uppercase tracking-[0.35em] text-zinc-500">Kagari</p>
          <h1 class="mt-3 text-4xl font-semibold tracking-tight sm:text-5xl">Service Status</h1>
          <p class="mt-3 max-w-2xl leading-7 text-zinc-400">公开展示经过脱敏处理的服务可用性与汇总指标。</p>
        </div>
        <button
          class="rounded-full border border-zinc-700 px-5 py-3 text-sm font-medium transition hover:border-zinc-300 disabled:cursor-wait disabled:opacity-60"
          type="button"
          :disabled="requestState === 'loading'"
          @click="refreshStatus"
        >
          {{ requestState === 'loading' ? '正在刷新' : '刷新状态' }}
        </button>
      </header>

      <p v-if="requestState === 'loading'" class="mt-8 text-zinc-400" role="status" aria-live="polite">正在获取最新服务状态。</p>
      <p v-else-if="requestState === 'idle'" class="mt-8 text-zinc-400" role="status">服务状态将在页面加载后更新。</p>
      <p v-else-if="requestState === 'error'" class="mt-8 rounded-xl border border-rose-400/40 bg-rose-400/10 p-4 text-rose-100" role="alert">{{ errorMessage }}</p>
      <p v-else-if="isDegraded" class="mt-8 rounded-xl border border-amber-400/40 bg-amber-400/10 p-4 text-amber-100" role="alert">部分服务处于降级状态，正在持续监测。</p>

      <section v-if="serviceStatus" class="mt-8 space-y-8" aria-label="服务状态详情">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          <article v-for="metric in resourceMetrics" :key="metric.label" class="rounded-2xl border border-zinc-800 bg-zinc-900/60 p-5">
            <h2 class="text-sm font-medium text-zinc-400">{{ metric.label }}</h2>
            <p class="mt-3 text-xl font-semibold text-zinc-100">{{ metric.value }}</p>
          </article>
        </div>

        <div class="grid gap-8 lg:grid-cols-2">
          <section aria-labelledby="container-status-heading">
            <h2 id="container-status-heading" class="text-xl font-semibold">容器</h2>
            <ul class="mt-4 space-y-3">
              <li v-for="container in serviceStatus.containers" :key="container.name" class="flex items-center justify-between rounded-xl border border-zinc-800 bg-zinc-900/60 px-5 py-4">
                <span>{{ container.name }}</span>
                <div class="flex gap-2">
                  <span class="rounded-full border px-3 py-1 text-sm" :class="stateClass(container.state)">{{ stateLabel(container.state) }}</span>
                  <span class="rounded-full border px-3 py-1 text-sm" :class="stateClass(container.resources)">资源 {{ stateLabel(container.resources) }}</span>
                </div>
              </li>
            </ul>
          </section>

          <section aria-labelledby="application-status-heading">
            <h2 id="application-status-heading" class="text-xl font-semibold">应用可用性</h2>
            <ul class="mt-4 space-y-3">
              <li v-for="application in serviceStatus.applications" :key="application.name" class="flex items-center justify-between rounded-xl border border-zinc-800 bg-zinc-900/60 px-5 py-4">
                <span>{{ application.name }}</span>
                <span class="rounded-full border px-3 py-1 text-sm" :class="stateClass(application.state)">{{ stateLabel(application.state) }}</span>
              </li>
            </ul>
          </section>
        </div>

        <p class="border-t border-zinc-800 pt-6 text-sm text-zinc-500">采样时间：{{ sampledAt }}</p>
      </section>
    </div>
  </main>
</template>
