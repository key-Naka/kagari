export default defineEventHandler((event) => {
  const runtimeConfig = useRuntimeConfig(event)
  const siteOrigin = new URL(runtimeConfig.public.siteUrl).origin
  setHeader(event, 'Content-Type', 'text/plain; charset=utf-8')
  return [
    'User-agent: *',
    'Allow: /',
    'Disallow: /admin',
    `Sitemap: ${siteOrigin}/sitemap.xml`,
    '',
  ].join('\n')
})
