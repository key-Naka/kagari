export default defineNuxtPlugin(() => {
  const router = useRouter()
  const transition = usePageTransition()

  router.beforeEach(async (to, from) => {
    if (to.fullPath === from.fullPath) {
      return true
    }
    if (to.path.startsWith('/admin') || from.path.startsWith('/admin')) {
      return true
    }
    window.dispatchEvent(new Event('kagari:navigation'))
    await transition.cover()
    return true
  })

  router.afterEach((_to, _from, failure) => {
    if (failure) {
      transition.reveal()
      return
    }
    nextTick(() => {
      requestAnimationFrame(() => transition.reveal())
    })
  })
})
