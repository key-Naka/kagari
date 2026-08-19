import { computed, toValue, type MaybeRefOrGetter } from 'vue'

interface PublicSeoOptions {
  title: MaybeRefOrGetter<string>
  description: MaybeRefOrGetter<string>
  image?: MaybeRefOrGetter<string | undefined>
  type?: 'website' | 'article'
}

export function usePublicSeo(options: PublicSeoOptions): void {
  const route = useRoute()
  const runtimeConfig = useRuntimeConfig()
  const siteOrigin = new URL(runtimeConfig.public.siteUrl).origin
  const canonical = computed(() => new URL(route.path, `${siteOrigin}/`).toString())
  const title = computed(() => toValue(options.title))
  const description = computed(() => toValue(options.description))
  const image = computed(() => (options.image ? toValue(options.image) : undefined) || new URL('/share-card.png', `${siteOrigin}/`).toString())

  useHead({
    link: [{ rel: 'canonical', href: canonical }],
  })
  useSeoMeta({
    title,
    description,
    ogTitle: title,
    ogDescription: description,
    ogUrl: canonical,
    ogType: options.type ?? 'website',
    ogImage: image,
    twitterCard: 'summary_large_image',
    twitterTitle: title,
    twitterDescription: description,
    twitterImage: image,
  })
}
