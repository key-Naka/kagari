<script setup lang="ts">
import { parseGalleryItems } from '~/utils/gallery-items'

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const { data: seededAlbumItems } = await useAsyncData('gallery-items', async () => {
  const items = parseGalleryItems(await $fetch<unknown>(`${apiBase}/api/v1/gallery-items`))
  if (!items) throw new Error('invalid gallery item response')
  return items
}, { default: () => [] })

function createSeedArtwork(colors: readonly [string, string, string]): string {
  const [shadow, middle, light] = colors
  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 600 760">
      <defs>
        <linearGradient id="field" x1="0" y1="0" x2="1" y2="1">
          <stop stop-color="${shadow}"/>
          <stop offset=".55" stop-color="${middle}"/>
          <stop offset="1" stop-color="${light}"/>
        </linearGradient>
        <radialGradient id="signal">
          <stop stop-color="${light}" stop-opacity=".9"/>
          <stop offset="1" stop-color="${light}" stop-opacity="0"/>
        </radialGradient>
        <pattern id="lines" width="54" height="54" patternUnits="userSpaceOnUse" patternTransform="rotate(28)">
          <path d="M0 0V54" stroke="#fff" stroke-opacity=".08"/>
        </pattern>
      </defs>
      <rect width="600" height="760" fill="url(#field)"/>
      <rect width="600" height="760" fill="url(#lines)"/>
      <circle cx="410" cy="210" r="190" fill="url(#signal)"/>
      <ellipse cx="300" cy="370" rx="210" ry="110" fill="none" stroke="#fff" stroke-opacity=".35" stroke-width="3" transform="rotate(-18 300 370)"/>
      <path d="M82 90h180M82 90v180M518 670H338M518 670V490" fill="none" stroke="#fff" stroke-opacity=".28" stroke-width="3"/>
    </svg>
  `.trim()
  return `data:image/svg+xml,${encodeURIComponent(svg)}`
}

const albumItems = computed(() => seededAlbumItems.value.map(item => ({
  ...item,
  src: item.imageUrl ?? createSeedArtwork(item.colors),
})))

usePublicSeo({
  title: '无界相册 · Kagari',
  description: '在 Kagari 的无限二维画布中拖拽探索有限的视觉残片。',
})
</script>

<template>
  <main class="gallery-page">
    <div class="gallery-page__grain" aria-hidden="true" />
    <div class="gallery-page__shell">
      <NuxtLink to="/" class="gallery-page__back cursor-target">← 返回首页</NuxtLink>

      <header class="gallery-page__header">
        <div>
          <p class="gallery-page__index">ARCHIVE / 05 · LIMINAL FIELD</p>
          <h1>无界相册</h1>
          <p class="gallery-page__intro">有限的视觉残片，被安置在一片没有尽头的二维场域。按住并向任意方向拖动，边界会将它们送回另一侧。</p>
        </div>
        <div class="gallery-page__summary" aria-label="相册摘要">
          <span>{{ String(albumItems.length).padStart(2, '0') }}</span>
          <small>FINITE ALBUM ITEMS</small>
          <i aria-hidden="true" />
        </div>
      </header>

      <div class="gallery-page__instructions">
        <p><span>01</span> 鼠标按住拖拽</p>
        <p><span>02</span> 单指自由穿行</p>
        <p><span>∞</span> 四边持续回绕</p>
      </div>

      <GalleryInfiniteGallery :items="albumItems" />

      <footer class="gallery-page__footer">
        <p>画布手势结束后，页面仍可继续滚动。</p>
        <NuxtLink to="/github" class="cursor-target">NEXT ARCHIVE · GITHUB →</NuxtLink>
      </footer>
    </div>
  </main>
</template>

<style scoped>
.gallery-page { position: relative; min-height: 140vh; overflow: hidden; background: radial-gradient(circle at 76% 8%, rgb(88 28 135 / 0.12), transparent 34rem), radial-gradient(circle at 9% 64%, rgb(67 56 202 / 0.07), transparent 30rem), #09090b; color: #f4f4f5; }
.gallery-page__grain { position: absolute; inset: 0; opacity: 0.18; background-image: repeating-linear-gradient(0deg, transparent, transparent 3px, rgb(255 255 255 / 0.018) 4px); pointer-events: none; }
.gallery-page__shell { position: relative; width: min(88rem, calc(100% - 2rem)); margin: 0 auto; padding: 3rem 0 9rem; }
.gallery-page__back { color: #a1a1aa; font-size: 0.78rem; letter-spacing: 0.08em; text-decoration: none; transition: color 160ms ease; }
.gallery-page__back:hover { color: #e4e4e7; }
.gallery-page__back:focus-visible, .gallery-page__footer a:focus-visible { outline: 2px solid #c4b5fd; outline-offset: 4px; }
.gallery-page__header { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: end; gap: 3rem; margin-top: 2.5rem; padding-bottom: 2.25rem; border-bottom: 1px solid #27272a; }
.gallery-page__index { color: #8b5cf6; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.64rem; letter-spacing: 0.28em; }
.gallery-page__header h1 { margin-top: 0.8rem; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(3.4rem, 9vw, 8rem); font-weight: 400; letter-spacing: -0.065em; line-height: 0.84; text-wrap: balance; }
.gallery-page__intro { max-width: 37rem; margin-top: 1.6rem; color: #a1a1aa; font-size: 0.94rem; line-height: 1.85; }
.gallery-page__summary { display: grid; min-width: 10rem; justify-items: end; }
.gallery-page__summary span { font-family: Georgia, 'Times New Roman', serif; font-size: 4.5rem; font-variant-numeric: tabular-nums; line-height: 0.8; }
.gallery-page__summary small { margin-top: 0.8rem; color: #a1a1aa; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.7rem; letter-spacing: 0.14em; }
.gallery-page__summary i { display: block; width: 4rem; height: 1px; margin-top: 1.3rem; background: linear-gradient(90deg, transparent, #8b5cf6); }
.gallery-page__instructions { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 0.75rem 2rem; padding: 1rem 0; color: #a1a1aa; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.7rem; letter-spacing: 0.1em; }
.gallery-page__instructions p { margin: 0; }
.gallery-page__instructions span { margin-right: 0.45rem; color: #8b5cf6; }
.gallery-page__footer { display: flex; align-items: center; justify-content: space-between; gap: 2rem; padding-top: 2.5rem; color: #a1a1aa; font-size: 0.78rem; letter-spacing: 0.06em; }
.gallery-page__footer a { color: #c4b5fd; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.72rem; letter-spacing: 0.12em; text-decoration: none; }
@media (max-width: 700px) { .gallery-page { min-height: 150vh; } .gallery-page__shell { padding-top: 2rem; } .gallery-page__header { grid-template-columns: 1fr; gap: 1.5rem; } .gallery-page__summary { display: none; } .gallery-page__intro { font-size: 0.86rem; } .gallery-page__instructions { justify-content: flex-start; } .gallery-page__footer { display: grid; } }
@media (prefers-reduced-motion: reduce) { .gallery-page__back { transition: none; } }
</style>
