export interface AdminMedia {
  id: number
  objectKey: string
  publicUrl: string
  kind: 'image' | 'audio'
  mimeType: string
  size: number
  originalName: string
  width?: number
  height?: number
  durationMs?: number
  createdAt: string
}

interface UploadCredentials {
  uploadToken: string
  uploadUrl: string
  objectKey: string
}

export function useAdminMediaUpload() {
  const { requestApi } = useAdminApi()

  async function imageMetadata(file: File): Promise<{ width: number, height: number }> {
    const objectUrl = URL.createObjectURL(file)
    const image = new Image()
    try {
      return await new Promise((resolve, reject) => {
        image.addEventListener('load', () => image.naturalWidth > 0 && image.naturalHeight > 0
          ? resolve({ width: image.naturalWidth, height: image.naturalHeight })
          : reject(new Error('无法识别图片尺寸。')), { once: true })
        image.addEventListener('error', () => reject(new Error('无法读取图片文件。')), { once: true })
        image.src = objectUrl
      })
    } finally {
      URL.revokeObjectURL(objectUrl)
    }
  }

  async function audioMetadata(file: File): Promise<{ durationMs: number }> {
    const objectUrl = URL.createObjectURL(file)
    const audio = new Audio()
    audio.preload = 'metadata'
    try {
      return await new Promise((resolve, reject) => {
        audio.addEventListener('loadedmetadata', () => Number.isFinite(audio.duration) && audio.duration > 0
          ? resolve({ durationMs: Math.round(audio.duration * 1000) })
          : reject(new Error('无法识别音频时长。')), { once: true })
        audio.addEventListener('error', () => reject(new Error('无法读取音频媒体元数据。')), { once: true })
        audio.src = objectUrl
        audio.load()
      })
    } finally {
      audio.removeAttribute('src')
      audio.load()
      URL.revokeObjectURL(objectUrl)
    }
  }

  async function uploadMedia(file: File, kind: 'image' | 'audio', onStage?: (stage: string) => void): Promise<AdminMedia> {
    const metadata = kind === 'image' ? await imageMetadata(file) : await audioMetadata(file)
    onStage?.('正在申请短期上传凭证…')
    const credentialsResponse = await requestApi('/api/v1/admin/media/upload-credentials', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ kind, mimeType: file.type, size: file.size, filename: file.name }),
    })
    const credentials = await credentialsResponse.json() as UploadCredentials

    onStage?.('正在直传七牛对象存储…')
    const upload = new FormData()
    upload.append('token', credentials.uploadToken)
    upload.append('key', credentials.objectKey)
    upload.append('file', file)
    const uploadResponse = await fetch(credentials.uploadUrl, { method: 'POST', body: upload })
    if (!uploadResponse.ok) throw new Error(`媒体直传失败（HTTP ${uploadResponse.status}）。`)

    onStage?.('正在登记媒体元数据…')
    const registrationResponse = await requestApi('/api/v1/admin/media', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        objectKey: credentials.objectKey,
        kind,
        mimeType: file.type,
        size: file.size,
        originalName: file.name,
        ...metadata,
      }),
    })
    return await registrationResponse.json() as AdminMedia
  }

  return { uploadMedia }
}
