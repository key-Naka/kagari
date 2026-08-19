<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

definePageMeta({ middleware: 'admin-auth' })

type ProjectStatus = 'draft' | 'published'

interface AdminProject {
  id: number
  title: string
  slug: string
  coverUrl: string
  description: string
  technologies: string[]
  types: string[]
  featured: boolean
  sortOrder: number
  status: ProjectStatus
  websiteUrl: string
  repositoryUrl: string
}

interface ProjectForm {
  title: string
  slug: string
  coverUrl: string
  description: string
  technologies: string
  types: string
  featured: boolean
  sortOrder: number
  status: ProjectStatus
  websiteUrl: string
  repositoryUrl: string
}

const { requestApi, redirectExpiredSession } = useAdminApi()
const projects = ref<AdminProject[]>([])
const { uploadMedia } = useAdminMediaUpload()
const activeProjectId = ref<number | null>(null)
const coverFile = ref<File | null>(null)
const coverInputKey = ref(0)
const form = ref<ProjectForm>(newProjectForm())
const isLoading = ref(true)
const isSaving = ref(false)
const deletingId = ref<number | null>(null)
const errorMessage = ref('')
const successMessage = ref('')
const uploadStage = ref('')
const isEditing = computed(() => activeProjectId.value !== null)

function newProjectForm(): ProjectForm {
  return {
    title: '',
    slug: '',
    coverUrl: '',
    description: '',
    technologies: '',
    types: '',
    featured: false,
    sortOrder: 0,
    status: 'draft',
    websiteUrl: '',
    repositoryUrl: '',
  }
}

function splitTags(value: string): string[] {
  return value.split(',').map(tag => tag.trim()).filter(Boolean)
}

async function loadProjects(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''
  try {
    const response = await requestApi('/api/v1/admin/projects')
    projects.value = await response.json() as AdminProject[]
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '无法加载作品。'
  } finally {
    isLoading.value = false
  }
}

function startCreate(): void {
  activeProjectId.value = null
  form.value = newProjectForm()
  coverFile.value = null
  coverInputKey.value += 1
  errorMessage.value = ''
  successMessage.value = ''
}

function startEdit(project: AdminProject): void {
  activeProjectId.value = project.id
  coverFile.value = null
  coverInputKey.value += 1
  form.value = {
    title: project.title,
    slug: project.slug,
    coverUrl: project.coverUrl,
    description: project.description,
    technologies: project.technologies.join(', '),
    types: project.types.join(', '),
    featured: project.featured,
    sortOrder: project.sortOrder,
    status: project.status,
    websiteUrl: project.websiteUrl,
    repositoryUrl: project.repositoryUrl,
  }
  errorMessage.value = ''
  successMessage.value = ''
}

async function saveProject(): Promise<void> {
  isSaving.value = true
  errorMessage.value = ''
  successMessage.value = ''
  const isCreating = activeProjectId.value === null
  const path = activeProjectId.value === null ? '/api/v1/admin/projects' : `/api/v1/admin/projects/${activeProjectId.value}`

  try {
    if (coverFile.value) {
      const media = await uploadMedia(coverFile.value, 'image', stage => { uploadStage.value = stage })
      form.value.coverUrl = media.publicUrl
    }
    const payload = {
      ...form.value,
      technologies: splitTags(form.value.technologies),
      types: splitTags(form.value.types),
    }
    await requestApi(path, {
      method: activeProjectId.value === null ? 'POST' : 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    await loadProjects()
    activeProjectId.value = null
    form.value = newProjectForm()
    coverFile.value = null
    coverInputKey.value += 1
    successMessage.value = isCreating
      ? payload.status === 'published' ? '作品已发布。' : '作品草稿已创建。'
      : '作品已更新。'
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '保存作品失败。'
  } finally {
    uploadStage.value = ''
    isSaving.value = false
  }
}

async function deleteProject(project: AdminProject): Promise<void> {
  if (!window.confirm(`永久删除“${project.title}”？此操作无法恢复。`)) {
    return
  }
  deletingId.value = project.id
  errorMessage.value = ''
  successMessage.value = ''
  try {
    await requestApi(`/api/v1/admin/projects/${project.id}`, {
      method: 'DELETE',
    })
    if (activeProjectId.value === project.id) {
      startCreate()
    }
    successMessage.value = '作品已永久删除。'
    await loadProjects()
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '删除作品失败。'
  } finally {
    deletingId.value = null
  }
}

onMounted(loadProjects)
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-4 py-10 text-zinc-100 sm:px-6 lg:px-8">
    <section class="mx-auto max-w-6xl">
      <NuxtLink to="/admin" class="text-sm text-zinc-500 transition hover:text-zinc-200">返回管理控制台</NuxtLink>

      <header class="mt-8 flex flex-wrap items-end justify-between gap-5 border-b border-zinc-800 pb-7">
        <div>
          <p class="text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">Administration / Works</p>
          <h1 class="mt-3 text-3xl font-semibold">作品管理</h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-zinc-400">维护公开作品的归档信息、发布状态与展示顺序。</p>
        </div>
        <button type="button" class="border border-zinc-700 px-4 py-2 text-sm font-medium transition hover:border-zinc-300" @click="startCreate">新建作品</button>
      </header>

      <p v-if="errorMessage" class="mt-6 border border-rose-400/40 bg-rose-400/10 p-4 text-sm text-rose-100" role="alert">{{ errorMessage }}</p>
      <p v-if="successMessage" class="mt-6 border border-emerald-400/40 bg-emerald-400/10 p-4 text-sm text-emerald-100" role="status">{{ successMessage }}</p>
      <p v-if="uploadStage" class="mt-6 border border-violet-400/40 bg-violet-400/10 p-4 text-sm text-violet-100" role="status">{{ uploadStage }}</p>

      <div class="mt-8 grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(22rem,0.85fr)]">
        <section aria-labelledby="project-list-heading">
          <div class="flex items-baseline justify-between border-b border-zinc-800 pb-3">
            <h2 id="project-list-heading" class="text-lg font-semibold">全部作品</h2>
            <span class="text-sm text-zinc-500">{{ projects.length }} 项</span>
          </div>
          <p v-if="isLoading" class="py-6 text-sm text-zinc-400" role="status">正在读取作品。</p>
          <p v-else-if="projects.length === 0" class="border-b border-zinc-800 py-8 text-sm text-zinc-400">还没有作品。先创建一份草稿。</p>
          <ol v-else class="divide-y divide-zinc-800">
            <li v-for="project in projects" :key="project.id" class="py-5">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <h3 class="text-lg font-semibold">{{ project.title }}</h3>
                    <span class="border px-2 py-1 text-xs" :class="project.status === 'published' ? 'border-emerald-400/40 text-emerald-200' : 'border-zinc-700 text-zinc-400'">{{ project.status === 'published' ? '已发布' : '草稿' }}</span>
                    <span v-if="project.featured" class="border border-amber-400/40 px-2 py-1 text-xs text-amber-200">精选</span>
                  </div>
                  <p class="mt-2 truncate text-sm text-zinc-500">/{{ project.slug }}</p>
                  <p class="mt-3 line-clamp-2 text-sm leading-6 text-zinc-400">{{ project.description }}</p>
                </div>
                <div class="flex shrink-0 gap-2">
                  <button type="button" class="border border-zinc-700 px-3 py-2 text-sm transition hover:border-zinc-300" @click="startEdit(project)">编辑</button>
                  <button type="button" class="border border-rose-400/40 px-3 py-2 text-sm text-rose-200 transition hover:border-rose-200 disabled:opacity-50" :disabled="deletingId === project.id" @click="deleteProject(project)">{{ deletingId === project.id ? '删除中' : '删除' }}</button>
                </div>
              </div>
            </li>
          </ol>
        </section>

        <section class="border border-zinc-800 bg-zinc-900/30 p-5 sm:p-6" aria-labelledby="project-form-heading">
          <div class="border-b border-zinc-800 pb-4">
            <p class="text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">{{ isEditing ? 'Edit entry' : 'New entry' }}</p>
            <h2 id="project-form-heading" class="mt-2 text-xl font-semibold">{{ isEditing ? '编辑作品' : '新建作品' }}</h2>
          </div>
          <form class="mt-6 space-y-5" @submit.prevent="saveProject">
            <label class="block text-sm font-medium">标题<input v-model.trim="form.title" required maxlength="160" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" /></label>
            <label class="block text-sm font-medium">稳定 slug<input v-model.trim="form.slug" required pattern="[a-z0-9]+(-[a-z0-9]+)*" maxlength="160" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 font-mono outline-none transition focus:border-emerald-300" /></label>
            <label class="block text-sm font-medium">封面 HTTPS 地址<input v-model.trim="form.coverUrl" :required="!coverFile" type="url" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" /></label>
            <label class="block text-sm font-medium">或上传封面图片<input :key="coverInputKey" type="file" accept="image/jpeg,image/png,image/webp,image/avif" class="mt-2 block w-full text-sm text-zinc-400 file:mr-4 file:border-0 file:bg-zinc-800 file:px-4 file:py-2 file:text-zinc-100" @change="coverFile = ($event.currentTarget as HTMLInputElement).files?.[0] ?? null" /></label>
            <label class="block text-sm font-medium">说明<textarea v-model.trim="form.description" required maxlength="6000" rows="5" class="mt-2 w-full resize-y border border-zinc-700 bg-zinc-950 px-3 py-2 leading-6 outline-none transition focus:border-emerald-300" /></label>
            <div class="grid gap-5 sm:grid-cols-2">
              <label class="block text-sm font-medium">技术标签<input v-model="form.technologies" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" placeholder="Go, Vue" /></label>
              <label class="block text-sm font-medium">类型标签<input v-model="form.types" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" placeholder="网站, 工具" /></label>
            </div>
            <div class="grid gap-5 sm:grid-cols-2">
              <label class="block text-sm font-medium">状态<select v-model="form.status" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300"><option value="draft">草稿</option><option value="published">已发布</option></select></label>
              <label class="block text-sm font-medium">排序<input v-model.number="form.sortOrder" min="0" type="number" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" /></label>
            </div>
            <label class="flex items-center gap-3 text-sm font-medium"><input v-model="form.featured" type="checkbox" class="h-4 w-4 accent-emerald-300" />作为精选作品</label>
            <label class="block text-sm font-medium">网站 HTTPS 地址<input v-model.trim="form.websiteUrl" type="url" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" /></label>
            <label class="block text-sm font-medium">源码或仓库 HTTPS 地址<input v-model.trim="form.repositoryUrl" type="url" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none transition focus:border-emerald-300" /></label>
            <div class="flex flex-wrap gap-3 border-t border-zinc-800 pt-5">
              <button type="submit" class="bg-emerald-300 px-4 py-2 text-sm font-semibold text-zinc-950 transition hover:bg-emerald-200 disabled:cursor-wait disabled:opacity-50" :disabled="isSaving">{{ isSaving ? '保存中' : isEditing ? '保存修改' : '创建作品' }}</button>
              <button v-if="isEditing" type="button" class="border border-zinc-700 px-4 py-2 text-sm transition hover:border-zinc-300" :disabled="isSaving" @click="startCreate">取消编辑</button>
            </div>
          </form>
        </section>
      </div>
    </section>
  </main>
</template>
