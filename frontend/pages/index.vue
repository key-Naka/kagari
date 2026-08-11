<template>
  <main class="min-h-screen bg-zinc-950 px-6 py-16 text-zinc-100 sm:px-10">
    <div class="mx-auto max-w-5xl">
      <p class="mb-4 text-sm uppercase tracking-[0.35em] text-zinc-500">Kagari</p>
      <h1 class="max-w-3xl text-5xl font-semibold tracking-tight sm:text-7xl">创作、文字与声音的个人档案。</h1>
      <p class="mt-8 max-w-2xl text-lg leading-8 text-zinc-400">一个面向公开内容与 Administration Console 的 Nuxt 4 SSR 基础站点。</p>
      <nav class="mt-12 flex flex-wrap gap-4 text-sm text-zinc-300" aria-label="主要导航">
        <NuxtLink class="rounded-full border border-zinc-700 px-5 py-3 hover:border-zinc-300" to="/works">作品</NuxtLink>
        <NuxtLink class="rounded-full border border-zinc-700 px-5 py-3 hover:border-zinc-300" to="/blog">博客</NuxtLink>
        <NuxtLink class="rounded-full border border-zinc-700 px-5 py-3 hover:border-zinc-300" to="/music">音乐</NuxtLink>
        <NuxtLink class="rounded-full border border-zinc-700 px-5 py-3 hover:border-zinc-300" to="/status">服务状态</NuxtLink>
        <NuxtLink class="rounded-full border border-zinc-700 px-5 py-3 hover:border-zinc-300" to="/admin">Administration Console</NuxtLink>
      </nav>
    </div>
  </main>
</template>
