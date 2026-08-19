type ApiErrorPayload = { error?: string }

export class AdminApiError extends Error {
  constructor(readonly status: number, message: string) {
    super(message)
  }
}

export function useAdminApi() {
  const runtimeConfig = useRuntimeConfig()
  const apiBase = runtimeConfig.public.apiBase.replace(/\/$/, '')

  async function responseError(response: Response): Promise<string> {
    const fallback = `请求失败（HTTP ${response.status}）。`
    try {
      const payload = await response.json() as ApiErrorPayload
      return payload.error || fallback
    } catch {
      return fallback
    }
  }

  async function requestApi(path: string, options: RequestInit = {}): Promise<Response> {
    try {
      const response = await fetch(`${apiBase}${path}`, { ...options, credentials: 'include' })
      if (!response.ok) throw new AdminApiError(response.status, await responseError(response))
      return response
    } catch (error) {
      if (error instanceof Error) throw error
      throw new Error('网络请求失败，请检查连接后重试。')
    }
  }

  async function redirectExpiredSession(error: unknown): Promise<boolean> {
    if (!(error instanceof AdminApiError) || error.status !== 401) return false
    await navigateTo('/admin/login?reason=session-expired')
    return true
  }

  return { requestApi, redirectExpiredSession }
}
