export interface GalleryItem {
  id: string
  title: string
  note: string
  alt: string
  year: string
  anchorX: number
  anchorY: number
  width: string
  aspectRatio: string
  colors: readonly [string, string, string]
}

type UnknownRecord = Record<string, unknown>

const colorPattern = /^#[0-9a-f]{6}$/i
const widthPattern = /^\d+(?:\.\d+)?vw$/
const aspectRatioPattern = /^\d+(?:\.\d+)? \/ \d+(?:\.\d+)?$/
const yearPattern = /^\d{4}$/

function isRecord(value: unknown): value is UnknownRecord {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function isAnchor(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= 1
}

function parseColors(value: unknown): GalleryItem['colors'] | null {
  if (!Array.isArray(value) || value.length !== 3 || !value.every(color => typeof color === 'string' && colorPattern.test(color))) return null
  return [value[0], value[1], value[2]]
}

function parseGalleryItem(value: unknown): GalleryItem | null {
  if (
    !isRecord(value)
    || typeof value.id !== 'string'
    || typeof value.title !== 'string'
    || typeof value.note !== 'string'
    || typeof value.alt !== 'string'
    || typeof value.year !== 'string'
    || !yearPattern.test(value.year)
    || !isAnchor(value.anchorX)
    || !isAnchor(value.anchorY)
    || typeof value.width !== 'string'
    || !widthPattern.test(value.width)
    || typeof value.aspectRatio !== 'string'
    || !aspectRatioPattern.test(value.aspectRatio)
  ) return null
  const colors = parseColors(value.colors)
  if (!colors) return null
  return {
    id: value.id,
    title: value.title,
    note: value.note,
    alt: value.alt,
    year: value.year,
    anchorX: value.anchorX,
    anchorY: value.anchorY,
    width: value.width,
    aspectRatio: value.aspectRatio,
    colors,
  }
}

export function parseGalleryItems(value: unknown): GalleryItem[] | null {
  if (!Array.isArray(value)) return null
  const items = value.map(parseGalleryItem)
  return items.every((item): item is GalleryItem => item !== null) ? items : null
}
