<script setup lang="ts">
import type { HomeArchiveModule } from '~/utils/home-archive'

const { module } = defineProps<{ module: HomeArchiveModule }>()
</script>

<template>
  <NuxtLink
    :to="module.route"
    :aria-label="`进入${module.label}档案`"
    class="archive-card cursor-target"
    :class="[`archive-card--${module.key}`, `archive-card--${module.state}`]"
    :data-module="module.key"
  >
    <div v-if="module.imageUrl" class="archive-card__media">
      <img :src="module.imageUrl" :alt="module.imageAlt" loading="lazy">
      <span class="archive-card__media-shade" aria-hidden="true" />
    </div>

    <div class="archive-card__frame">
      <header class="archive-card__header">
        <span class="archive-card__index">{{ module.index }}</span>
        <span class="archive-card__eyebrow">{{ module.eyebrow }}</span>
        <span class="archive-card__state" aria-hidden="true" />
      </header>

      <div class="archive-card__body">
        <p class="archive-card__metric">{{ module.metric }}</p>
        <div>
          <p class="archive-card__label">{{ module.label }}</p>
          <h3>{{ module.headline }}</h3>
          <p class="archive-card__detail">{{ module.detail }}</p>
        </div>
      </div>

      <footer class="archive-card__footer">
        <span>{{ module.meta }}</span>
        <span class="archive-card__enter" aria-hidden="true">ENTER ARCHIVE ↗</span>
      </footer>
    </div>
  </NuxtLink>
</template>

<style scoped>
.archive-card { position: relative; display: block; min-height: 22rem; overflow: hidden; border: 1px solid rgb(82 82 91 / 0.68); color: #f4f4f5; background: rgb(9 9 11 / 0.88); text-decoration: none; isolation: isolate; transition: border-color 240ms ease, transform 240ms ease, background 240ms ease; }
.archive-card::before { position: absolute; z-index: 2; inset: 0; border: 1px solid transparent; background: linear-gradient(135deg, rgb(196 181 253 / 0.2), transparent 34%, transparent 70%, rgb(217 191 129 / 0.12)) border-box; content: ''; opacity: 0; pointer-events: none; transition: opacity 240ms ease; mask: linear-gradient(#000 0 0) padding-box, linear-gradient(#000 0 0); mask-composite: exclude; }
.archive-card:hover, .archive-card:focus-visible { border-color: rgb(196 181 253 / 0.74); background: rgb(18 17 23 / 0.94); outline: none; transform: translateY(-3px); }
.archive-card:hover::before, .archive-card:focus-visible::before { opacity: 1; }
.archive-card__media { position: absolute; z-index: -1; inset: 0; }
.archive-card__media img { width: 100%; height: 100%; color: transparent; object-fit: cover; font-size: 0; filter: saturate(0.72) contrast(1.08); opacity: 0.48; transform: scale(1.01); transition: filter 500ms ease, opacity 500ms ease, transform 700ms cubic-bezier(0.16, 1, 0.3, 1); }
.archive-card:hover .archive-card__media img, .archive-card:focus-visible .archive-card__media img { filter: saturate(0.95) contrast(1.04); opacity: 0.62; transform: scale(1.045); }
.archive-card__media-shade { position: absolute; inset: 0; background: linear-gradient(90deg, rgb(9 9 11 / 0.95), rgb(9 9 11 / 0.7) 56%, rgb(9 9 11 / 0.24)), linear-gradient(0deg, rgb(9 9 11 / 0.9), transparent 58%); }
.archive-card__frame { display: flex; min-height: inherit; flex-direction: column; padding: 1.35rem; }
.archive-card__header { display: grid; grid-template-columns: auto 1fr auto; align-items: center; gap: 0.8rem; color: #8f8f99; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.61rem; letter-spacing: 0.17em; }
.archive-card__index { color: #c4b5fd; }
.archive-card__state { width: 0.42rem; height: 0.42rem; border-radius: 999px; background: #86efac; box-shadow: 0 0 0.7rem rgb(134 239 172 / 0.42); }
.archive-card--empty .archive-card__state { background: #a1a1aa; box-shadow: none; }
.archive-card--unavailable .archive-card__state { background: #fbbf24; box-shadow: 0 0 0.7rem rgb(251 191 36 / 0.3); }
.archive-card__body { display: grid; flex: 1; grid-template-columns: auto minmax(0, 1fr); align-items: end; gap: clamp(1rem, 4vw, 3rem); padding: 3.5rem 0 2.5rem; }
.archive-card__metric { align-self: start; margin: 0; color: rgb(244 244 245 / 0.17); font-family: Georgia, 'Times New Roman', serif; font-size: clamp(4rem, 8vw, 7.5rem); font-variant-numeric: tabular-nums; letter-spacing: -0.08em; line-height: 0.72; }
.archive-card__label { margin: 0 0 0.7rem; color: #d9bf81; font-size: 0.66rem; letter-spacing: 0.24em; text-transform: uppercase; }
.archive-card h3 { max-width: 30rem; margin: 0; font-family: Georgia, 'Times New Roman', serif; font-size: clamp(1.7rem, 3.3vw, 3.35rem); font-weight: 400; letter-spacing: -0.035em; line-height: 1.02; text-wrap: balance; }
.archive-card__detail { max-width: 32rem; margin: 1rem 0 0; color: #a1a1aa; font-size: 0.84rem; line-height: 1.75; }
.archive-card__footer { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding-top: 1rem; border-top: 1px solid rgb(82 82 91 / 0.52); color: #71717a; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.58rem; letter-spacing: 0.12em; }
.archive-card__enter { color: #c4b5fd; transform: translateX(-0.25rem); transition: color 180ms ease, transform 180ms ease; }
.archive-card:hover .archive-card__enter, .archive-card:focus-visible .archive-card__enter { color: #ede9fe; transform: translateX(0); }
.archive-card--status, .archive-card--visitor-messages { min-height: 18rem; }
.archive-card--status .archive-card__body, .archive-card--visitor-messages .archive-card__body { padding-block: 2.2rem; }
@media (max-width: 650px) { .archive-card { min-height: 20rem; } .archive-card__frame { padding: 1.1rem; } .archive-card__eyebrow { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; } .archive-card__body { grid-template-columns: 1fr; gap: 1.5rem; padding: 2.6rem 0 2rem; } .archive-card__metric { font-size: 4.25rem; } .archive-card__footer { align-items: flex-end; } }
@media (prefers-reduced-motion: reduce) { .archive-card, .archive-card::before, .archive-card__media img, .archive-card__enter { transition: none; } .archive-card:hover, .archive-card:focus-visible { transform: none; } }
</style>
