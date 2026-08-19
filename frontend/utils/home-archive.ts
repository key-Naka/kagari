export type HomeSummaryState = 'ready' | 'empty' | 'unavailable'

export interface HomeArchiveSignal {
  state: HomeSummaryState
  metric: string
  headline: string
  detail: string
  meta: string
  imageUrl?: string
  imageAlt?: string
}

export interface HomeArchiveData {
  works: HomeArchiveSignal
  blog: HomeArchiveSignal
  music: HomeArchiveSignal
  github: HomeArchiveSignal
  gallery: HomeArchiveSignal
  status: HomeArchiveSignal
  visitorMessages: HomeArchiveSignal
}

export interface HomeArchiveModule extends HomeArchiveSignal {
  key: 'works' | 'blog' | 'music' | 'github' | 'gallery' | 'status' | 'visitor-messages'
  index: string
  label: string
  route: string
  eyebrow: string
}

type PublicFetcher = (url: string) => Promise<unknown>
type UnknownRecord = Record<string, unknown>
type Availability = 'operational' | 'degraded' | 'unavailable'
type HomeModuleKey = keyof HomeArchiveData

interface CollectionSummary<T> {
  availability: Availability
  count: number
  item?: T
}

interface ProjectSummary {
  title: string
  coverUrl: string
  description: string
  featured: boolean
}

interface PostSummary {
  title: string
  summary: string
  publishedAt: string
}

interface TrackSummary {
  title: string
  coverUrl: string
}

interface GitHubSummary {
  repository: string
  description: string
}

interface VisitorMessageSummary {
  nickname: string
  content: string
}

interface GallerySummary {
  availability: Availability
  count: number
}

interface StatusSummary {
  availability: Availability
  operational: number
  total: number
}

const unavailableHeadlines: Record<HomeModuleKey, string> = {
  works: '作品摘要暂时离线',
  blog: '博客摘要暂时离线',
  music: '音乐摘要暂时离线',
  github: 'GitHub 摘要暂时离线',
  gallery: '相册摘要暂时离线',
  status: '状态摘要暂时离线',
  visitorMessages: '留言摘要暂时离线',
}

function isRecord(value: unknown): value is UnknownRecord {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function isAvailability(value: unknown): value is Availability {
  return value === 'operational' || value === 'degraded' || value === 'unavailable'
}

function isCount(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0
}

function parseCollection<T>(value: unknown, parseItem: (value: unknown) => T | null): CollectionSummary<T> | null {
  if (!isRecord(value) || !isAvailability(value.availability) || !isCount(value.count)) return null
  if (value.item === undefined || value.item === null) {
    return { availability: value.availability, count: value.count }
  }
  const item = parseItem(value.item)
  return item ? { availability: value.availability, count: value.count, item } : null
}

function parseProject(value: unknown): ProjectSummary | null {
  if (
    !isRecord(value)
    || typeof value.title !== 'string'
    || typeof value.coverUrl !== 'string'
    || typeof value.description !== 'string'
    || typeof value.featured !== 'boolean'
  ) return null
  return { title: value.title, coverUrl: value.coverUrl, description: value.description, featured: value.featured }
}

function parsePost(value: unknown): PostSummary | null {
  if (!isRecord(value) || typeof value.title !== 'string' || typeof value.summary !== 'string' || typeof value.publishedAt !== 'string') return null
  return { title: value.title, summary: value.summary, publishedAt: value.publishedAt }
}

function parseTrack(value: unknown): TrackSummary | null {
  if (!isRecord(value) || typeof value.title !== 'string' || typeof value.coverUrl !== 'string') return null
  return { title: value.title, coverUrl: value.coverUrl }
}

function parseGitHub(value: unknown): GitHubSummary | null {
  if (!isRecord(value) || typeof value.repository !== 'string' || typeof value.description !== 'string') return null
  return { repository: value.repository, description: value.description }
}

function parseVisitorMessage(value: unknown): VisitorMessageSummary | null {
  if (!isRecord(value) || typeof value.nickname !== 'string' || typeof value.content !== 'string') return null
  return { nickname: value.nickname, content: value.content }
}

function parseGallery(value: unknown): GallerySummary | null {
  if (!isRecord(value) || !isAvailability(value.availability) || !isCount(value.count)) return null
  return { availability: value.availability, count: value.count }
}

function parseStatus(value: unknown): StatusSummary | null {
  if (
    !isRecord(value)
    || !isAvailability(value.availability)
    || !isCount(value.operational)
    || !isCount(value.total)
    || value.operational > value.total
  ) return null
  return { availability: value.availability, operational: value.operational, total: value.total }
}

function unavailable(key: HomeModuleKey): HomeArchiveSignal {
  return {
    state: 'unavailable',
    metric: '—',
    headline: unavailableHeadlines[key],
    detail: '公开 API 暂时没有回应，独立档案仍可继续访问。',
    meta: 'SIGNAL INTERRUPTED',
  }
}

export function createUnavailableHomeArchive(): HomeArchiveData {
  return {
    works: unavailable('works'),
    blog: unavailable('blog'),
    music: unavailable('music'),
    github: unavailable('github'),
    gallery: unavailable('gallery'),
    status: unavailable('status'),
    visitorMessages: unavailable('visitorMessages'),
  }
}

function summarizeProjects(summary: CollectionSummary<ProjectSummary> | null): HomeArchiveSignal {
  if (!summary || summary.availability === 'unavailable') return unavailable('works')
  if (!summary.item) return { state: 'empty', metric: '00', headline: '作品档案正在编目', detail: '首个 Portfolio Project 发布后会在这里留下索引。', meta: 'NO PUBLISHED PROJECTS' }
  return { state: 'ready', metric: String(summary.count).padStart(2, '0'), headline: summary.item.title, detail: summary.item.description, meta: summary.item.featured ? 'FEATURED PORTFOLIO PROJECT' : 'LATEST PORTFOLIO PROJECT', imageUrl: summary.item.coverUrl, imageAlt: `${summary.item.title} 封面` }
}

function summarizePosts(summary: CollectionSummary<PostSummary> | null): HomeArchiveSignal {
  if (!summary || summary.availability === 'unavailable') return unavailable('blog')
  if (!summary.item) return { state: 'empty', metric: '00', headline: '尚无公开文章', detail: '新的 Blog Post 会从这里进入时间归档。', meta: 'WRITING ARCHIVE EMPTY' }
  return { state: 'ready', metric: String(summary.count).padStart(2, '0'), headline: summary.item.title, detail: summary.item.summary, meta: `LATEST / ${summary.item.publishedAt.slice(0, 10)}` }
}

function summarizeTracks(summary: CollectionSummary<TrackSummary> | null): HomeArchiveSignal {
  if (!summary || summary.availability === 'unavailable') return unavailable('music')
  if (!summary.item) return { state: 'empty', metric: '00', headline: '音乐档案尚未启用', detail: '第一条 Track 启用后，声音会从这里开始。', meta: 'NO ENABLED TRACKS' }
  return { state: 'ready', metric: String(summary.count).padStart(2, '0'), headline: summary.item.title, detail: '一段被保存下来的声响。进入独立音乐页后由访客主动播放。', meta: 'ENABLED TRACK', imageUrl: summary.item.coverUrl, imageAlt: `${summary.item.title} 封面` }
}

function summarizeGitHub(summary: CollectionSummary<GitHubSummary> | null): HomeArchiveSignal {
  if (!summary || summary.availability === 'unavailable') return unavailable('github')
  if (!summary.item) return { state: 'empty', metric: '00', headline: '公开活动仍在同步', detail: 'Contribution Heatmap 和仓库索引暂时没有公开记录。', meta: 'PUBLIC ACTIVITY EMPTY' }
  return { state: 'ready', metric: String(summary.count).padStart(2, '0'), headline: summary.item.repository, detail: summary.item.description || '最近的公开工程活动已经写入快照。', meta: summary.availability === 'operational' ? 'LIVE PUBLIC SNAPSHOT' : 'LAST SUCCESSFUL SNAPSHOT' }
}

function summarizeGallery(summary: GallerySummary | null): HomeArchiveSignal {
  if (!summary || summary.availability === 'unavailable') return unavailable('gallery')
  if (summary.count === 0) return { state: 'empty', metric: '00', headline: '等待第一张 Album Item', detail: '视觉残片加入公开相册后，会从这里进入无界画布。', meta: 'VISUAL ARCHIVE EMPTY' }
  return { state: 'ready', metric: '∞', headline: `${summary.count} 个 Album Item`, detail: '有限的视觉残片在二维画布中四边回绕；拖拽越过边界，它们会从另一侧归来。', meta: 'FINITE NODES / ENDLESS FIELD' }
}

function summarizeStatus(summary: StatusSummary | null): HomeArchiveSignal {
  if (!summary || summary.availability === 'unavailable') return unavailable('status')
  if (summary.total === 0) return { state: 'empty', metric: '00', headline: '等待第一轮状态采样', detail: '公开服务检查尚未写入摘要。', meta: 'SERVICE STATUS EMPTY' }
  const allOperational = summary.availability === 'operational' && summary.operational === summary.total
  return { state: 'ready', metric: allOperational ? 'OK' : '△', headline: allOperational ? '全部服务正常' : '部分服务正在降级', detail: `${summary.operational} / ${summary.total} 项公开检查正常`, meta: 'SANITIZED SERVICE STATUS' }
}

function summarizeVisitorMessages(summary: CollectionSummary<VisitorMessageSummary> | null): HomeArchiveSignal {
  if (!summary || summary.availability === 'unavailable') return unavailable('visitorMessages')
  if (!summary.item) return { state: 'empty', metric: '00', headline: '等待第一条 Visitor Message', detail: '匿名访客或使用昵称的访客可以在独立留言板留下公开讯号。', meta: 'PUBLIC SIGNALS EMPTY' }
  return { state: 'ready', metric: String(summary.count).padStart(2, '0'), headline: summary.item.content, detail: summary.item.nickname ? `来自 ${summary.item.nickname} 的公开讯号。` : '来自匿名访客的公开讯号。', meta: 'LATEST VISITOR MESSAGE' }
}

function summarizeHomeArchive(value: unknown): HomeArchiveData {
  if (!isRecord(value)) return createUnavailableHomeArchive()
  return {
    works: summarizeProjects(parseCollection(value.works, parseProject)),
    blog: summarizePosts(parseCollection(value.blog, parsePost)),
    music: summarizeTracks(parseCollection(value.music, parseTrack)),
    github: summarizeGitHub(parseCollection(value.github, parseGitHub)),
    gallery: summarizeGallery(parseGallery(value.gallery)),
    status: summarizeStatus(parseStatus(value.status)),
    visitorMessages: summarizeVisitorMessages(parseCollection(value.visitorMessages, parseVisitorMessage)),
  }
}

export async function loadHomeArchive(apiBase: string, fetcher: PublicFetcher): Promise<HomeArchiveData> {
  try {
    return summarizeHomeArchive(await fetcher(`${apiBase}/api/v1/home`))
  } catch {
    return createUnavailableHomeArchive()
  }
}
