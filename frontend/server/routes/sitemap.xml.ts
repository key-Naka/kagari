interface SlugRecord {
  slug: string
}

const staticPublicRoutes = [
  '/',
  '/works',
  '/blog',
  '/music',
  '/github',
  '/gallery',
  '/status',
  '/visitor-messages',
]

export default defineEventHandler(async (event) => {
  const runtimeConfig = useRuntimeConfig(event)
  const siteOrigin = new URL(runtimeConfig.public.siteUrl).origin
  const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')
  const routes = new Set(staticPublicRoutes)

  const [projects, posts] = await Promise.allSettled([
    $fetch<unknown>(`${apiBase}/api/v1/projects`),
    $fetch<unknown>(`${apiBase}/api/v1/posts`),
  ])
  if (projects.status === 'fulfilled') {
    addContentRoutes(routes, '/works', projects.value)
  }
  if (posts.status === 'fulfilled') {
    addContentRoutes(routes, '/blog', posts.value)
  }

  const locations = [...routes]
    .map(path => `  <url><loc>${escapeXml(new URL(path, `${siteOrigin}/`).toString())}</loc></url>`)
    .join('\n')

  setHeader(event, 'Content-Type', 'application/xml; charset=utf-8')
  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${locations}\n</urlset>\n`
})

function addContentRoutes(routes: Set<string>, prefix: string, value: unknown): void {
  if (!Array.isArray(value)) {
    return
  }
  for (const item of value) {
    if (isSlugRecord(item)) {
      routes.add(`${prefix}/${encodeURIComponent(item.slug)}`)
    }
  }
}

function isSlugRecord(value: unknown): value is SlugRecord {
  return value !== null
    && typeof value === 'object'
    && 'slug' in value
    && typeof value.slug === 'string'
    && value.slug.length > 0
}

function escapeXml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&apos;')
}
