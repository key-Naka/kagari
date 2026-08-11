/**
 * 验证管理会话：受保护页面仅允许已认证用户访问，登录页则拒绝已认证用户重复登录。
 */
export default defineNuxtRouteMiddleware(async (to) => {
  const runtimeConfig = useRuntimeConfig()
  const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')

  try {
    const response = await fetch(`${apiBase}/api/v1/admin/session`, {
      credentials: 'include',
    })

    if (response.ok) {
      if (to.path === '/admin/login') {
        return navigateTo('/admin')
      }

      return
    }
  } catch {
    // 会话检查不可用时不能放行受保护页面；登录页保留为可见错误提交入口。
  }

  if (to.path !== '/admin/login') {
    return navigateTo('/admin/login')
  }
})
