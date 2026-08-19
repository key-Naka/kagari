<script setup lang="ts">
import { computed, onMounted, ref, shallowRef } from 'vue'
import type { AdminMedia } from '~/composables/useAdminMediaUpload'

definePageMeta({ middleware: 'admin-auth' })

type ImageMedia = AdminMedia & { kind: 'image', width: number, height: number }

interface AdminAlbumItem {
  id: number
  title: string
  note: string
  alt: string
  year: string
  image: ImageMedia | null
  anchorX: number
  anchorY: number
  width: string
  aspectRatio: string
  colors: [string, string, string]
  published: boolean
  sortOrder: number
}

interface AlbumItemForm {
  title: string
  note: string
  alt: string
  year: string
  anchorX: number
  anchorY: number
  width: string
  aspectRatio: string
  colors: [string, string, string]
  published: boolean
  sortOrder: number
}

const { requestApi, redirectExpiredSession } = useAdminApi()
const { uploadMedia } = useAdminMediaUpload()
const items = shallowRef<AdminAlbumItem[]>([])
const activeItemId = shallowRef<number | null>(null)
const imageFile = shallowRef<File | null>(null)
const fileInputKey = shallowRef(0)
const form = ref<AlbumItemForm>(newAlbumItemForm())
const isLoading = shallowRef(true)
const isSaving = shallowRef(false)
const deletingId = shallowRef<number | null>(null)
const uploadStage = shallowRef('')
const errorMessage = shallowRef('')
const successMessage = shallowRef('')

const isEditing = computed(() => activeItemId.value !== null)
const activeItem = computed(() => items.value.find(item => item.id === activeItemId.value) ?? null)

function newAlbumItemForm(): AlbumItemForm {
  return {
    title: '',
    note: '',
    alt: '',
    year: String(new Date().getFullYear()),
    anchorX: 0.5,
    anchorY: 0.5,
    width: '12vw',
    aspectRatio: '4 / 5',
    colors: ['#100f18', '#352157', '#9f7aea'],
    published: false,
    sortOrder: 0,
  }
}

async function loadItems(): Promise<void> {
  isLoading.value = true
  errorMessage.value = ''
  try {
    const response = await requestApi('/api/v1/admin/gallery-items')
    items.value = await response.json() as AdminAlbumItem[]
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '无法读取 Album Item。'
  } finally {
    isLoading.value = false
  }
}

function startCreate(): void {
  activeItemId.value = null
  imageFile.value = null
  fileInputKey.value += 1
  form.value = newAlbumItemForm()
  errorMessage.value = ''
  successMessage.value = ''
}

function startEdit(item: AdminAlbumItem): void {
  activeItemId.value = item.id
  imageFile.value = null
  fileInputKey.value += 1
  form.value = {
    title: item.title,
    note: item.note,
    alt: item.alt,
    year: item.year,
    anchorX: item.anchorX,
    anchorY: item.anchorY,
    width: item.width,
    aspectRatio: item.aspectRatio,
    colors: [...item.colors],
    published: item.published,
    sortOrder: item.sortOrder,
  }
  errorMessage.value = ''
  successMessage.value = ''
}

function selectImage(event: Event): void {
  imageFile.value = (event.currentTarget as HTMLInputElement).files?.[0] ?? null
}

async function saveItem(): Promise<void> {
  const wasCreating = activeItemId.value === null
  errorMessage.value = ''
  successMessage.value = ''
  isSaving.value = true
  try {
    const image = imageFile.value
      ? await uploadMedia(imageFile.value, 'image', stage => { uploadStage.value = stage }) as ImageMedia
      : activeItem.value?.image
    if (!image) throw new Error('请选择 Album Item 图片。')
    uploadStage.value = '正在保存 Album Item…'
    const path = activeItemId.value === null
      ? '/api/v1/admin/gallery-items'
      : `/api/v1/admin/gallery-items/${activeItemId.value}`
    await requestApi(path, {
      method: activeItemId.value === null ? 'POST' : 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...form.value, imageMediaId: image.id }),
    })
    await loadItems()
    startCreate()
    successMessage.value = wasCreating ? 'Album Item 已创建。' : 'Album Item 已更新。'
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '保存 Album Item 失败。'
  } finally {
    isSaving.value = false
    uploadStage.value = ''
  }
}

async function deleteItem(item: AdminAlbumItem): Promise<void> {
  if (!window.confirm(`永久删除“${item.title}”？此操作无法恢复。`)) return
  deletingId.value = item.id
  errorMessage.value = ''
  successMessage.value = ''
  try {
    await requestApi(`/api/v1/admin/gallery-items/${item.id}`, { method: 'DELETE' })
    if (activeItemId.value === item.id) startCreate()
    await loadItems()
    successMessage.value = 'Album Item 已永久删除。'
  } catch (error) {
    if (await redirectExpiredSession(error)) return
    errorMessage.value = error instanceof Error ? error.message : '删除 Album Item 失败。'
  } finally {
    deletingId.value = null
  }
}

onMounted(loadItems)
</script>

<template>
  <main class="min-h-screen bg-zinc-950 px-4 py-10 text-zinc-100 sm:px-6 lg:px-8">
    <section class="mx-auto max-w-7xl">
      <header class="flex flex-wrap items-end justify-between gap-4 border-b border-zinc-800 pb-6">
        <div>
          <p class="text-xs font-medium uppercase tracking-[0.24em] text-violet-400">Administration / Liminal Field</p>
          <h1 class="mt-3 text-3xl font-semibold">Album Item 管理</h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 text-zinc-400">上传图片、配置二维画布位置，并控制公开相册中的发布状态与顺序。</p>
        </div>
        <div class="flex gap-3">
          <NuxtLink to="/admin" class="border border-zinc-700 px-4 py-2 text-sm transition hover:border-zinc-300">返回管理控制台</NuxtLink>
          <button type="button" class="border border-zinc-700 px-4 py-2 text-sm transition hover:border-zinc-300" @click="startCreate">新建 Album Item</button>
        </div>
      </header>

      <p v-if="errorMessage" class="mt-6 border border-rose-400/40 bg-rose-400/10 p-4 text-sm text-rose-100" role="alert">{{ errorMessage }}</p>
      <p v-if="successMessage" class="mt-6 border border-emerald-400/40 bg-emerald-400/10 p-4 text-sm text-emerald-100" role="status">{{ successMessage }}</p>
      <p v-if="uploadStage" class="mt-6 border border-violet-400/40 bg-violet-400/10 p-4 text-sm text-violet-100" role="status">{{ uploadStage }}</p>

      <div class="mt-8 grid gap-8 lg:grid-cols-[minmax(0,0.85fr)_minmax(28rem,1.15fr)]">
        <section aria-labelledby="album-item-list-heading" class="border border-zinc-800 bg-zinc-900/30 p-5">
          <div class="flex items-baseline justify-between border-b border-zinc-800 pb-3">
            <h2 id="album-item-list-heading" class="text-lg font-semibold">全部 Album Item</h2>
            <span class="text-sm text-zinc-500">{{ items.length }} 项</span>
          </div>
          <p v-if="isLoading" class="py-8 text-sm text-zinc-400" role="status">正在读取 Album Item…</p>
          <p v-else-if="items.length === 0" class="py-8 text-sm text-zinc-400">还没有 Album Item。先上传第一张图片。</p>
          <ul v-else class="divide-y divide-zinc-800">
            <li v-for="item in items" :key="item.id" class="grid grid-cols-[5rem_minmax(0,1fr)] gap-4 py-5">
              <img v-if="item.image" :src="item.image.publicUrl" :alt="item.alt" class="aspect-square w-20 object-cover">
              <div v-else class="aspect-square w-20 border border-zinc-700" :style="{ background: `linear-gradient(135deg, ${item.colors.join(', ')})` }" role="img" :aria-label="item.alt" />
              <div class="min-w-0">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <h3 class="truncate font-medium">{{ item.title }}</h3>
                    <p class="mt-1 text-xs text-zinc-500">{{ item.year }} · 排序 {{ item.sortOrder }} · {{ item.published ? '已发布' : '草稿' }}</p>
                  </div>
                  <span class="rounded-full px-2 py-1 text-xs" :class="item.published ? 'bg-emerald-400/10 text-emerald-300' : 'bg-zinc-800 text-zinc-400'">{{ item.published ? '公开' : '未公开' }}</span>
                </div>
                <div class="mt-4 flex flex-wrap gap-2">
                  <button type="button" class="border border-zinc-700 px-3 py-2 text-sm transition hover:border-zinc-300" @click="startEdit(item)">编辑</button>
                  <button type="button" class="border border-rose-400/40 px-3 py-2 text-sm text-rose-200 transition hover:border-rose-200 disabled:opacity-50" :disabled="deletingId === item.id" @click="deleteItem(item)">{{ deletingId === item.id ? '删除中' : '永久删除' }}</button>
                </div>
              </div>
            </li>
          </ul>
        </section>

        <section aria-labelledby="album-item-form-heading" class="border border-zinc-800 bg-zinc-900/30 p-5">
          <p class="text-xs font-medium uppercase tracking-[0.24em] text-zinc-500">{{ isEditing ? 'Edit fragment' : 'New fragment' }}</p>
          <h2 id="album-item-form-heading" class="mt-2 text-xl font-semibold">{{ isEditing ? '编辑 Album Item' : '新建 Album Item' }}</h2>
          <form class="mt-6 space-y-5" @submit.prevent="saveItem">
            <div class="grid gap-5 sm:grid-cols-2">
              <label class="block text-sm font-medium">标题<input v-model.trim="form.title" required maxlength="160" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none focus:border-violet-300"></label>
              <label class="block text-sm font-medium">年份<input v-model.trim="form.year" required inputmode="numeric" pattern="\d{4}" maxlength="4" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none focus:border-violet-300"></label>
            </div>
            <label class="block text-sm font-medium">档案注记<input v-model.trim="form.note" required maxlength="255" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none focus:border-violet-300"></label>
            <label class="block text-sm font-medium">图片替代文本<textarea v-model.trim="form.alt" required maxlength="500" rows="3" class="mt-2 w-full resize-y border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none focus:border-violet-300" /></label>
            <label class="block text-sm font-medium">图片文件<input :key="fileInputKey" type="file" accept="image/jpeg,image/png,image/webp,image/avif" :required="!activeItem?.image" class="mt-2 block w-full text-sm text-zinc-400 file:mr-4 file:border-0 file:bg-zinc-800 file:px-3 file:py-2 file:text-zinc-100" @change="selectImage"><small v-if="activeItem?.image" class="mt-2 block break-all text-zinc-500">当前：{{ activeItem.image.originalName }}（选择新文件即可替换）</small><small v-else-if="isEditing" class="mt-2 block text-zinc-500">此迁移条目尚无媒体文件，保存前必须上传图片。</small></label>
            <div class="grid gap-5 sm:grid-cols-2">
              <label class="block text-sm font-medium">水平锚点（0–1）<input v-model.number="form.anchorX" required min="0" max="1" step="0.01" type="number" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none focus:border-violet-300"></label>
              <label class="block text-sm font-medium">垂直锚点（0–1）<input v-model.number="form.anchorY" required min="0" max="1" step="0.01" type="number" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none focus:border-violet-300"></label>
              <label class="block text-sm font-medium">画布宽度<input v-model.trim="form.width" required pattern="\d+(\.\d+)?vw" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none focus:border-violet-300" placeholder="12vw"></label>
              <label class="block text-sm font-medium">宽高比<input v-model.trim="form.aspectRatio" required pattern="\d+(\.\d+)? / \d+(\.\d+)?" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none focus:border-violet-300" placeholder="4 / 5"></label>
            </div>
            <fieldset>
              <legend class="text-sm font-medium">视觉色组</legend>
              <div class="mt-2 grid grid-cols-3 gap-3">
                <label v-for="(_color, index) in form.colors" :key="index" class="text-xs text-zinc-400">颜色 {{ index + 1 }}<input v-model="form.colors[index]" required pattern="#[0-9a-fA-F]{6}" class="mt-1 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 font-mono text-sm text-zinc-100 outline-none focus:border-violet-300"></label>
              </div>
            </fieldset>
            <div class="grid gap-5 sm:grid-cols-2">
              <label class="block text-sm font-medium">排序<input v-model.number="form.sortOrder" required min="0" type="number" class="mt-2 w-full border border-zinc-700 bg-zinc-950 px-3 py-2 outline-none focus:border-violet-300"></label>
              <label class="flex items-end gap-3 pb-2 text-sm font-medium"><input v-model="form.published" type="checkbox" class="h-4 w-4 accent-violet-400">公开发布</label>
            </div>
            <div class="flex flex-wrap gap-3 border-t border-zinc-800 pt-5">
              <button type="submit" class="bg-violet-200 px-4 py-2 text-sm font-semibold text-zinc-950 hover:bg-violet-100 disabled:cursor-wait disabled:opacity-50" :disabled="isSaving">{{ isSaving ? '处理中…' : isEditing ? '保存修改' : '创建 Album Item' }}</button>
              <button v-if="isEditing" type="button" class="border border-zinc-700 px-4 py-2 text-sm hover:border-zinc-300" :disabled="isSaving" @click="startCreate">取消编辑</button>
            </div>
          </form>
        </section>
      </div>
    </section>
  </main>
</template>
