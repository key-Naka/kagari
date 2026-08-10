export const useSiteStore = defineStore('site', () => {
  const navigation = [
    { label: '首页', to: '/' },
    { label: '作品', to: '/works' },
    { label: '博客', to: '/blog' },
    { label: '音乐', to: '/music' },
    { label: '相册', to: '/gallery' },
    { label: 'GitHub', to: '/github' },
  ]

  return { navigation }
})
