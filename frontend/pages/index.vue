<script setup lang="ts">
import { computed } from 'vue'
import { createUnavailableHomeArchive, loadHomeArchive, type HomeArchiveModule } from '~/utils/home-archive'

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
const defaultSiteConfig = {
  siteTitle: 'Kagari · 全栈工程师与独立创作者',
  seoSummary: 'Kagari 的作品导向首页：浏览全栈工程、Blog Post、Track、GitHub、相册、服务状态与 Visitor Message 档案。',
  shareImageUrl: '',
}
const { data: siteConfig } = await useAsyncData('public-site-config', async () => {
  try {
    return await $fetch<typeof defaultSiteConfig>(`${apiBase}/api/v1/site-config`)
  } catch {
    return defaultSiteConfig
  }
}, { default: () => defaultSiteConfig })
const { data, status, refresh } = await useAsyncData('home-archive', () => (
  loadHomeArchive(apiBase, url => $fetch<unknown>(url))
), {
  default: createUnavailableHomeArchive,
})

const isRefreshing = computed(() => status.value === 'pending')
const archiveModules = computed<HomeArchiveModule[]>(() => [
  { key: 'works', index: '01', label: '作品', route: '/works', eyebrow: 'SELECTED SYSTEMS', ...data.value.works },
  { key: 'blog', index: '02', label: '博客', route: '/blog', eyebrow: 'WRITTEN RECORDS', ...data.value.blog },
  { key: 'music', index: '03', label: '音乐', route: '/music', eyebrow: 'RESONANT OBJECTS', ...data.value.music },
  { key: 'github', index: '04', label: 'GitHub', route: '/github', eyebrow: 'PUBLIC ENGINEERING', ...data.value.github },
  { key: 'gallery', index: '05', label: '相册', route: '/gallery', eyebrow: 'LIMINAL FIELD', ...data.value.gallery },
  { key: 'status', index: '06', label: '服务状态', route: '/status', eyebrow: 'PUBLIC TELEMETRY', ...data.value.status },
  { key: 'visitor-messages', index: '07', label: '访客留言', route: '/visitor-messages', eyebrow: 'OPEN SIGNALS', ...data.value.visitorMessages },
])

usePublicSeo({
  title: () => siteConfig.value.siteTitle,
  description: () => siteConfig.value.seoSummary,
  image: () => siteConfig.value.shareImageUrl || undefined,
})
</script>

<template>
  <main class="home-page">
    <div class="home-page__grain" aria-hidden="true" />
    <div class="home-page__shell">
      <header class="home-hero">
        <div class="home-hero__ledger">
          <p>ARCHIVE MONOLITH</p>
          <span>YK / 2026</span>
        </div>

        <div class="home-hero__body">
          <div>
            <p class="home-hero__identity">全栈工程师 / 独立创作者</p>
            <h1>把系统、界面与声音，<em>归入同一座档案。</em></h1>
          </div>
          <div class="home-hero__statement">
            <p>我是 Kagari。设计并构建从 Go 服务、数据边界到 Nuxt 界面的完整系统，也保存写作、音乐与视觉实验。</p>
            <a href="#archive-index" class="cursor-target">浏览精选索引 <span aria-hidden="true">↓</span></a>
          </div>
        </div>

        <div class="home-hero__footer">
          <p><span>07</span> PUBLIC ARCHIVES</p>
          <p><span>SSR</span> LIVE API SUMMARIES</p>
          <p><span>CN</span> SINGLE AUTHOR</p>
        </div>
      </header>

      <section id="archive-index" class="home-index" aria-labelledby="archive-index-title">
        <header class="home-index__header">
          <div>
            <p>CURATED ENTRY POINTS / 01—07</p>
            <h2 id="archive-index-title">精选档案索引</h2>
          </div>
          <button type="button" class="home-index__refresh cursor-target" :disabled="isRefreshing" @click="refresh()">
            <span :class="{ 'home-index__refresh-mark--active': isRefreshing }" aria-hidden="true">↻</span>
            {{ isRefreshing ? '正在同步' : '同步摘要' }}
          </button>
        </header>

        <div class="home-index__grid">
          <HomeArchiveCard
            v-for="module in archiveModules"
            :key="module.key"
            :module="module"
            :class="`home-index__module--${module.key}`"
          />
        </div>
      </section>

      <footer class="home-footer">
        <p>每张卡片只保留一条线索；完整内容始终属于它自己的独立路由。</p>
        <NuxtLink to="/visitor-messages" class="cursor-target">留下公开讯号 →</NuxtLink>
      </footer>
    </div>
  </main>
</template>

<style scoped>
.home-page { position: relative; min-height: 100vh; overflow: hidden; color: #f4f4f5; background: radial-gradient(circle at 77% 5%, rgb(91 33 182 / 0.14), transparent 34rem), radial-gradient(circle at 12% 38%, rgb(217 191 129 / 0.045), transparent 28rem), #09090b; }
.home-page__grain { position: absolute; inset: 0; opacity: 0.2; background-image: repeating-linear-gradient(0deg, transparent, transparent 3px, rgb(255 255 255 / 0.018) 4px); pointer-events: none; }
.home-page__shell { position: relative; width: min(88rem, calc(100% - 2.5rem)); margin: 0 auto; padding: 4rem 0 6rem; }
.home-hero { min-height: calc(100vh - 8rem); display: flex; flex-direction: column; justify-content: space-between; padding: 1.2rem 0 2rem; }
.home-hero__ledger, .home-hero__footer, .home-index__header, .home-footer { display: flex; align-items: center; justify-content: space-between; gap: 1.5rem; }
.home-hero__ledger { color: #71717a; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.62rem; letter-spacing: 0.22em; }
.home-hero__ledger p, .home-hero__ledger span, .home-hero__footer p { margin: 0; }
.home-hero__body { display: grid; grid-template-columns: minmax(0, 1.65fr) minmax(17rem, 0.55fr); align-items: end; gap: clamp(3rem, 8vw, 8rem); padding: 7rem 0 5rem; }
.home-hero__identity { margin: 0 0 1.5rem; color: #c4b5fd; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.68rem; letter-spacing: 0.2em; }
.home-hero h1 { max-width: 62rem; margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(4rem, 9.2vw, 9.2rem); font-weight: 400; letter-spacing: -0.075em; line-height: 0.82; text-wrap: balance; }
.home-hero h1 em { display: block; color: #9f8dc6; font-style: italic; font-weight: 400; }
.home-hero__statement { padding-left: 1.5rem; border-left: 1px solid #3f3f46; }
.home-hero__statement p { margin: 0; color: #a1a1aa; font-size: 0.9rem; line-height: 1.9; }
.home-hero__statement a { display: flex; align-items: center; justify-content: space-between; margin-top: 2rem; padding: 0.9rem 0; border-bottom: 1px solid #52525b; color: #e4e4e7; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.67rem; letter-spacing: 0.16em; text-decoration: none; transition: border-color 180ms ease, color 180ms ease; }
.home-hero__statement a:hover, .home-hero__statement a:focus-visible { border-color: #c4b5fd; color: #c4b5fd; outline: none; }
.home-hero__footer { padding-top: 1rem; border-top: 1px solid #27272a; color: #71717a; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.58rem; letter-spacing: 0.12em; }
.home-hero__footer span { margin-right: 0.45rem; color: #d9bf81; }
.home-index { scroll-margin-top: 6rem; padding-top: 7rem; }
.home-index__header { align-items: end; padding-bottom: 1.7rem; border-bottom: 1px solid #3f3f46; }
.home-index__header p { margin: 0 0 0.8rem; color: #8b5cf6; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.62rem; letter-spacing: 0.2em; }
.home-index__header h2 { margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(2.6rem, 5vw, 4.8rem); font-weight: 400; letter-spacing: -0.055em; line-height: 0.9; }
.home-index__refresh { display: flex; align-items: center; gap: 0.65rem; border: 1px solid #52525b; padding: 0.75rem 1rem; color: #d4d4d8; background: rgb(9 9 11 / 0.62); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.65rem; letter-spacing: 0.1em; cursor: pointer; transition: border-color 180ms ease, color 180ms ease; }
.home-index__refresh:hover, .home-index__refresh:focus-visible { border-color: #c4b5fd; color: #ede9fe; outline: none; }
.home-index__refresh:disabled { cursor: wait; opacity: 0.6; }
.home-index__refresh-mark--active { display: inline-block; animation: home-refresh-spin 700ms linear infinite; }
.home-index__grid { display: grid; grid-template-columns: repeat(12, minmax(0, 1fr)); gap: 0.75rem; padding-top: 0.75rem; }
.home-index__module--works { grid-column: span 8; }
.home-index__module--blog { grid-column: span 4; }
.home-index__module--music, .home-index__module--github, .home-index__module--gallery { grid-column: span 4; }
.home-index__module--status, .home-index__module--visitor-messages { grid-column: span 6; }
.home-footer { margin-top: 6rem; padding-top: 1.5rem; border-top: 1px solid #27272a; color: #71717a; font-size: 0.76rem; letter-spacing: 0.04em; }
.home-footer p { margin: 0; }
.home-footer a { color: #c4b5fd; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.66rem; letter-spacing: 0.12em; text-decoration: none; }
.home-footer a:hover, .home-footer a:focus-visible { color: #ede9fe; outline: 1px solid #c4b5fd; outline-offset: 4px; }
@keyframes home-refresh-spin { to { transform: rotate(360deg); } }
@media (max-width: 980px) { .home-hero { min-height: auto; } .home-hero__body { grid-template-columns: 1fr; align-items: start; padding: 8rem 0 4rem; } .home-hero__statement { max-width: 34rem; } .home-index__module--works, .home-index__module--blog, .home-index__module--music, .home-index__module--github, .home-index__module--gallery, .home-index__module--status, .home-index__module--visitor-messages { grid-column: span 6; } }
@media (max-width: 650px) { .home-page__shell { width: min(100% - 1.25rem, 88rem); padding-top: 2rem; } .home-hero__ledger { align-items: flex-start; } .home-hero__body { gap: 2.5rem; padding: 6rem 0 3rem; } .home-hero h1 { font-size: clamp(3.4rem, 18vw, 5.7rem); } .home-hero__statement { padding-left: 1rem; } .home-hero__footer { display: grid; grid-template-columns: 1fr 1fr; } .home-hero__footer p:last-child { display: none; } .home-index { padding-top: 5rem; } .home-index__header { align-items: flex-start; } .home-index__refresh { flex: 0 0 auto; padding-inline: 0.8rem; } .home-index__grid { grid-template-columns: 1fr; } .home-index__module--works, .home-index__module--blog, .home-index__module--music, .home-index__module--github, .home-index__module--gallery, .home-index__module--status, .home-index__module--visitor-messages { grid-column: 1; } .home-footer { display: grid; } }
@media (prefers-reduced-motion: reduce) { .home-hero__statement a, .home-index__refresh { transition: none; } .home-index__refresh-mark--active { animation: none; } }
</style>
