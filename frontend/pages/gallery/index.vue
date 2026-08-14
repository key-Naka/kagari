<script setup lang="ts">
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

const seededAlbumItems = [
  { id: 'A-01', title: 'Violet Wake', note: 'afterimage / 00:14', alt: '紫色光晕穿过深色轨道的抽象影像', year: '2026', anchorX: 0.04, anchorY: 0.08, width: '12vw', aspectRatio: '4 / 5', colors: ['#100f18', '#352157', '#9f7aea'] },
  { id: 'A-02', title: 'Ash Meridian', note: 'north line / 04:20', alt: '灰白地平线与黑色天幕形成的抽象影像', year: '2025', anchorX: 0.27, anchorY: 0.16, width: '15vw', aspectRatio: '3 / 2', colors: ['#111113', '#39373b', '#b9a78c'] },
  { id: 'A-03', title: 'Nocturne Gate', note: 'threshold / 11:03', alt: '幽蓝门廊悬浮于夜色中的抽象影像', year: '2026', anchorX: 0.54, anchorY: 0.05, width: '11vw', aspectRatio: '1 / 1', colors: ['#071014', '#153647', '#5c90a8'] },
  { id: 'A-04', title: 'Ember Index', note: 'catalogue / 09:17', alt: '暗红余烬在黑色档案纹理中发光的抽象影像', year: '2024', anchorX: 0.79, anchorY: 0.2, width: '13vw', aspectRatio: '4 / 5', colors: ['#160c0f', '#53212a', '#c17362'] },
  { id: 'A-05', title: 'Silver Static', note: 'signal loss / 02:41', alt: '银色静电划过墨黑表面的抽象影像', year: '2025', anchorX: 0.98, anchorY: 0.04, width: '10vw', aspectRatio: '3 / 4', colors: ['#0b0c0f', '#292d35', '#a7adba'] },
  { id: 'A-06', title: 'Moss Reliquary', note: 'sealed / 18:08', alt: '墨绿光斑围绕古老容器的抽象影像', year: '2023', anchorX: 0.12, anchorY: 0.53, width: '14vw', aspectRatio: '3 / 2', colors: ['#09100d', '#17372c', '#6e9d78'] },
  { id: 'A-07', title: 'Quiet Orbit', note: 'drift / 07:33', alt: '苍白轨道环绕紫黑核心的抽象影像', year: '2026', anchorX: 0.39, anchorY: 0.48, width: '12vw', aspectRatio: '1 / 1', colors: ['#0f0c16', '#302248', '#8b73aa'] },
  { id: 'A-08', title: 'Cinder Bloom', note: 'specimen / 12:56', alt: '金色余烬如花朵在暗处展开的抽象影像', year: '2024', anchorX: 0.68, anchorY: 0.56, width: '16vw', aspectRatio: '16 / 10', colors: ['#130f09', '#4a311b', '#c59a59'] },
  { id: 'A-09', title: 'Veiled Current', note: 'channel / 03:02', alt: '青蓝水流被薄雾遮蔽的抽象影像', year: '2025', anchorX: 0.93, anchorY: 0.47, width: '11vw', aspectRatio: '4 / 5', colors: ['#071113', '#15333a', '#5e9295'] },
  { id: 'A-10', title: 'Obsidian Choir', note: 'resonance / 21:10', alt: '多重紫色回声穿过黑曜石平面的抽象影像', year: '2026', anchorX: 0.02, anchorY: 0.92, width: '11vw', aspectRatio: '1 / 1', colors: ['#0b0910', '#24152f', '#765189'] },
  { id: 'A-11', title: 'Pale Ceremony', note: 'room nine / 05:44', alt: '象牙白光柱排列成仪式空间的抽象影像', year: '2023', anchorX: 0.48, anchorY: 0.9, width: '13vw', aspectRatio: '3 / 2', colors: ['#11100f', '#45403a', '#bdb19f'] },
  { id: 'A-12', title: 'Last Signal', note: 'terminal / 23:59', alt: '玫红信号在深夜地平线上消退的抽象影像', year: '2025', anchorX: 0.98, anchorY: 0.94, width: '12vw', aspectRatio: '4 / 5', colors: ['#130a10', '#401b31', '#a85b83'] },
] as const

const albumItems = seededAlbumItems.map(item => ({
  ...item,
  src: createSeedArtwork(item.colors),
}))

useSeoMeta({
  title: '无界相册 · Kagari',
  description: '在 Kagari 的无限二维画布中拖拽探索有限的视觉残片。',
  ogTitle: '无界相册 · Kagari',
  ogDescription: '拖拽穿行于四边无缝回绕的 Album Item 档案。',
  twitterCard: 'summary_large_image',
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
