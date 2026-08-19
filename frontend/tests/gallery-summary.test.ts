import { describe, expect, it } from 'vitest'
import { parseGalleryItems } from '../utils/gallery-items'

describe('Gallery 公开数据契约', () => {
  it('只接受完整且安全的 Album Item 数组', () => {
    const item = {
      id: 'A-01', title: 'Violet Wake', note: 'afterimage', alt: '抽象影像', year: '2026',
      imageUrl: 'https://cdn.example.com/media/image/2026/08/violet.webp',
      anchorX: 0.04, anchorY: 0.08, width: '12vw', aspectRatio: '4 / 5',
      colors: ['#100f18', '#352157', '#9f7aea'],
    }
    expect(parseGalleryItems([item])).toEqual([item])
    expect(parseGalleryItems([{ ...item, anchorX: 'invalid' }])).toBeNull()
    expect(parseGalleryItems([{ ...item, imageUrl: 'javascript:alert(1)' }])).toBeNull()
    const { imageUrl: _imageUrl, ...seededItem } = item
    expect(parseGalleryItems([seededItem])).toEqual([seededItem])
  })
})
