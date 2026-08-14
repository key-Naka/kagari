<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, shallowRef } from 'vue'

interface AlbumItem {
  id: string
  title: string
  note: string
  src: string
  alt: string
  year: string
  anchorX: number
  anchorY: number
  width: string
  aspectRatio: string
  colors: readonly [string, string, string]
}

interface PositionedAlbumItem {
  element: HTMLElement
  x: number
  y: number
  width: number
  height: number
}

const { items } = defineProps<{
  items: readonly AlbumItem[]
}>()

const canvas = shallowRef<HTMLElement | null>(null)
const isReady = shallowRef(false)
const isDragging = shallowRef(false)
let activePointerId: number | null = null
let lastPointer = { x: 0, y: 0 }
let positionedItems: PositionedAlbumItem[] = []
let canvasWidth = 0
let canvasHeight = 0
let resizeObserver: ResizeObserver | null = null

function wrapCoordinate(value: number, boundary: number, itemSize: number): number {
  const period = boundary + itemSize
  return ((value + itemSize) % period + period) % period - itemSize
}

function paintPositions(): void {
  for (const item of positionedItems) {
    item.element.style.transform = `translate3d(${item.x}px, ${item.y}px, 0)`
  }
}

function measureCanvas(): void {
  const element = canvas.value
  if (!element) return

  const previousWidth = canvasWidth
  const previousHeight = canvasHeight
  canvasWidth = element.clientWidth
  canvasHeight = element.clientHeight
  if (canvasWidth === 0 || canvasHeight === 0) return

  const elements = Array.from(element.querySelectorAll<HTMLElement>('[data-gallery-item]'))
  positionedItems = elements.map((albumElement, index) => {
    const definition = items[index]!
    const width = albumElement.offsetWidth
    const height = albumElement.offsetHeight
    const previous = positionedItems[index]
    if (previous && previousWidth > 0 && previousHeight > 0) {
      return {
        element: albumElement,
        x: wrapCoordinate(
          ((previous.x + previous.width / 2) / previousWidth) * canvasWidth - width / 2,
          canvasWidth,
          width,
        ),
        y: wrapCoordinate(
          ((previous.y + previous.height / 2) / previousHeight) * canvasHeight - height / 2,
          canvasHeight,
          height,
        ),
        width,
        height,
      }
    }
    return {
      element: albumElement,
      x: definition.anchorX * canvasWidth - width / 2,
      y: definition.anchorY * canvasHeight - height / 2,
      width,
      height,
    }
  })
  paintPositions()
  isReady.value = true
}

function moveItems(deltaX: number, deltaY: number): void {
  for (const item of positionedItems) {
    item.x = wrapCoordinate(item.x + deltaX, canvasWidth, item.width)
    item.y = wrapCoordinate(item.y + deltaY, canvasHeight, item.height)
  }
  paintPositions()
}

function handlePointerDown(event: PointerEvent): void {
  if (!event.isPrimary || (event.pointerType === 'mouse' && event.button !== 0)) return
  const element = canvas.value
  if (!element) return

  activePointerId = event.pointerId
  lastPointer = { x: event.clientX, y: event.clientY }
  isDragging.value = true
  if (element.isConnected) {
    try {
      element.setPointerCapture(event.pointerId)
    } catch {
      // Synthetic PointerEvents do not participate in browser pointer capture.
    }
  }
}

function handlePointerMove(event: PointerEvent): void {
  if (activePointerId !== event.pointerId) return
  if (event.pointerType === 'touch') event.preventDefault()

  const deltaX = event.clientX - lastPointer.x
  const deltaY = event.clientY - lastPointer.y
  lastPointer = { x: event.clientX, y: event.clientY }
  moveItems(deltaX, deltaY)
}

function endPointer(event: PointerEvent): void {
  if (activePointerId !== event.pointerId) return
  const element = canvas.value
  if (element?.hasPointerCapture(event.pointerId)) {
    element.releasePointerCapture(event.pointerId)
  }
  activePointerId = null
  isDragging.value = false
}

onMounted(async () => {
  await nextTick()
  measureCanvas()
  if (canvas.value) {
    resizeObserver = new ResizeObserver(measureCanvas)
    resizeObserver.observe(canvas.value)
  }
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  resizeObserver = null
  activePointerId = null
})
</script>

<template>
  <section class="infinite-gallery" aria-labelledby="gallery-canvas-title">
    <div class="infinite-gallery__rail" aria-hidden="true">
      <span>POINTER FIELD</span>
      <span>DRAG · WANDER · RETURN</span>
    </div>
    <div
      ref="canvas"
      class="infinite-gallery__canvas cursor-target"
      :class="{ 'infinite-gallery__canvas--dragging': isDragging }"
      :data-ready="isReady"
      data-testid="gallery-canvas"
      role="group"
      aria-describedby="gallery-canvas-instructions"
      @pointerdown="handlePointerDown"
      @pointermove="handlePointerMove"
      @pointerup="endPointer"
      @pointercancel="endPointer"
      @lostpointercapture="endPointer"
    >
      <h2 id="gallery-canvas-title" class="sr-only">可拖拽 Album Item 画布</h2>
      <p id="gallery-canvas-instructions" class="sr-only">按住鼠标或使用单指向任意方向拖动，Album Item 会从画布对侧回绕。</p>
      <div class="infinite-gallery__axis infinite-gallery__axis--horizontal" aria-hidden="true" />
      <div class="infinite-gallery__axis infinite-gallery__axis--vertical" aria-hidden="true" />

      <article
        v-for="item in items"
        :key="item.id"
        class="album-item"
        :style="{
          '--album-width': item.width,
          '--album-ratio': item.aspectRatio,
          '--album-color-a': item.colors[0],
          '--album-color-b': item.colors[1],
          '--album-color-c': item.colors[2],
        }"
        :data-album-id="item.id"
        data-gallery-item
        data-testid="album-item"
      >
        <div class="album-item__frame">
          <div class="album-item__image" role="img" :aria-label="item.alt">
            <img :src="item.src" alt="" draggable="false">
            <span class="album-item__flare" aria-hidden="true" />
            <span class="album-item__orbit" aria-hidden="true" />
          </div>
          <footer class="album-item__caption">
            <span>{{ item.id }}</span>
            <strong>{{ item.title }}</strong>
            <small>{{ item.year }}</small>
          </footer>
        </div>
        <p>{{ item.note }}</p>
      </article>
    </div>
    <div class="infinite-gallery__rail infinite-gallery__rail--footer" aria-hidden="true">
      <span>FINITE OBJECTS / INFINITE PASSAGE</span>
      <span>{{ String(items.length).padStart(2, '0') }} NODES</span>
    </div>
  </section>
</template>

<style scoped>
.infinite-gallery { border: 1px solid rgb(63 63 70 / 0.85); background: rgb(9 9 11 / 0.82); box-shadow: 0 2rem 8rem rgb(0 0 0 / 0.45); }
.infinite-gallery__rail { display: flex; min-height: 2.5rem; align-items: center; justify-content: space-between; gap: 1rem; border-bottom: 1px solid #27272a; padding: 0.65rem 0.9rem; color: #71717a; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.58rem; letter-spacing: 0.2em; }
.infinite-gallery__rail--footer { border-top: 1px solid #27272a; border-bottom: 0; color: #52525b; }
.infinite-gallery__canvas { position: relative; height: clamp(34rem, 72vh, 48rem); overflow: hidden; cursor: grab; touch-action: none; user-select: none; isolation: isolate; background: radial-gradient(circle at 50% 48%, rgb(88 28 135 / 0.11), transparent 42%), linear-gradient(rgb(255 255 255 / 0.025) 1px, transparent 1px), linear-gradient(90deg, rgb(255 255 255 / 0.025) 1px, transparent 1px), #09090b; background-size: auto, 3rem 3rem, 3rem 3rem, auto; }
.infinite-gallery__canvas::before, .infinite-gallery__canvas::after { position: absolute; z-index: 4; width: 2.2rem; height: 2.2rem; border-color: rgb(139 92 246 / 0.75); pointer-events: none; content: ''; }
.infinite-gallery__canvas::before { top: 0; left: 0; border-top: 1px solid; border-left: 1px solid; }
.infinite-gallery__canvas::after { right: 0; bottom: 0; border-right: 1px solid; border-bottom: 1px solid; }
.infinite-gallery__canvas--dragging { cursor: grabbing; }
.infinite-gallery__axis { position: absolute; z-index: -1; background: linear-gradient(90deg, transparent, rgb(167 139 250 / 0.16), transparent); pointer-events: none; }
.infinite-gallery__axis--horizontal { top: 50%; right: 0; left: 0; height: 1px; }
.infinite-gallery__axis--vertical { top: 0; bottom: 0; left: 50%; width: 1px; background: linear-gradient(transparent, rgb(167 139 250 / 0.14), transparent); }
.album-item { position: absolute; top: 0; left: 0; z-index: 2; width: clamp(8.5rem, var(--album-width), 15rem); will-change: transform; }
.album-item__frame { border: 1px solid rgb(228 228 231 / 0.25); background: #111114; box-shadow: 0 1.25rem 3rem rgb(0 0 0 / 0.42); }
.album-item__image { position: relative; overflow: hidden; aspect-ratio: var(--album-ratio); background: linear-gradient(143deg, var(--album-color-a), var(--album-color-b) 52%, var(--album-color-c)); }
.album-item__image img { width: 100%; height: 100%; object-fit: cover; pointer-events: none; }
.album-item__image::before, .album-item__image::after, .album-item__flare, .album-item__orbit { z-index: 1; }
.album-item__image::before { position: absolute; inset: 0; opacity: 0.44; background: repeating-linear-gradient(115deg, transparent 0 1.3rem, rgb(255 255 255 / 0.07) 1.35rem 1.4rem), linear-gradient(180deg, transparent 58%, rgb(0 0 0 / 0.64)); mix-blend-mode: screen; content: ''; }
.album-item__image::after { position: absolute; inset: 10%; border: 1px solid rgb(255 255 255 / 0.16); clip-path: polygon(0 0, 35% 0, 35% 1px, 1px 1px, 1px 35%, 0 35%, 0 0, 100% 0, 100% 35%, calc(100% - 1px) 35%, calc(100% - 1px) 1px, 65% 1px, 65% 0, 100% 0, 100% 100%, 65% 100%, 65% calc(100% - 1px), calc(100% - 1px) calc(100% - 1px), calc(100% - 1px) 65%, 100% 65%, 100% 100%, 0 100%, 0 65%, 1px 65%, 1px calc(100% - 1px), 35% calc(100% - 1px), 35% 100%, 0 100%); content: ''; }
.album-item__flare { position: absolute; top: 18%; left: 16%; width: 32%; aspect-ratio: 1; border-radius: 50%; background: var(--album-color-c); filter: blur(1.8rem); opacity: 0.42; }
.album-item__orbit { position: absolute; top: 25%; left: 25%; width: 52%; aspect-ratio: 1; border: 1px solid rgb(255 255 255 / 0.3); border-radius: 50%; box-shadow: 0 0 0 0.75rem rgb(255 255 255 / 0.025), 0 0 0 2.25rem rgb(255 255 255 / 0.018); transform: rotate(-18deg) scaleY(0.58); }
.album-item__caption { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: 0.55rem; border-top: 1px solid rgb(228 228 231 / 0.16); padding: 0.62rem 0.7rem; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.album-item__caption span, .album-item__caption small { color: #a1a1aa; font-size: 0.68rem; letter-spacing: 0.08em; }
.album-item__caption strong { overflow: hidden; color: #d4d4d8; font-family: Georgia, 'Times New Roman', serif; font-size: 0.76rem; font-weight: 400; letter-spacing: 0.03em; text-overflow: ellipsis; white-space: nowrap; }
.album-item > p { margin: 0.45rem 0 0 0.15rem; color: #a1a1aa; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.75rem; letter-spacing: 0.08em; text-transform: uppercase; }
@media (max-width: 640px) { .infinite-gallery__canvas { height: min(36rem, 68vh); min-height: 28rem; background-size: auto, 2rem 2rem, 2rem 2rem, auto; } .infinite-gallery__rail { letter-spacing: 0.11em; } .album-item { width: clamp(7rem, calc(var(--album-width) * 1.65), 10.5rem); } .album-item > p { display: none; } }
@media (prefers-reduced-motion: reduce) { .album-item { will-change: auto; } }
</style>
