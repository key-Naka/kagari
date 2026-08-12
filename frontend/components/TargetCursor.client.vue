<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, shallowRef, useTemplateRef } from 'vue'

type Gsap = typeof import('gsap')['gsap']
type GsapTween = ReturnType<Gsap['to']>
type GsapTimeline = ReturnType<Gsap['timeline']>

interface Point {
  x: number
  y: number
}

type CornerName = 'top-left' | 'top-right' | 'bottom-right' | 'bottom-left'

const CORNER_GAP = 3
const FREE_RADIUS = 18
const CORNERS = [
  { name: 'top-left', markClass: 'target-cursor__corner-mark--top-left' },
  { name: 'top-right', markClass: 'target-cursor__corner-mark--top-right' },
  { name: 'bottom-right', markClass: 'target-cursor__corner-mark--bottom-right' },
  { name: 'bottom-left', markClass: 'target-cursor__corner-mark--bottom-left' },
] as const satisfies ReadonlyArray<{ name: CornerName; markClass: string }>
const FREE_POSITIONS: Record<CornerName, Point> = {
  'top-left': { x: -FREE_RADIUS, y: -FREE_RADIUS },
  'top-right': { x: FREE_RADIUS, y: -FREE_RADIUS },
  'bottom-right': { x: FREE_RADIUS, y: FREE_RADIUS },
  'bottom-left': { x: -FREE_RADIUS, y: FREE_RADIUS },
}

const cursor = useTemplateRef<HTMLElement>('cursor')
const ring = useTemplateRef<HTMLElement>('ring')
const dot = useTemplateRef<HTMLElement>('dot')
const cornerElements = useTemplateRef<HTMLElement[]>('corner')
const enabled = shallowRef(false)
const locked = shallowRef(false)

let gsap: Gsap | null = null
let spinTween: GsapTween | null = null
let cornerTimeline: GsapTimeline | null = null
let moveCursorX: ((value: number) => void) | null = null
let moveCursorY: ((value: number) => void) | null = null
let lockedTarget: HTMLElement | null = null
let targetResizeObserver: ResizeObserver | null = null
let finePointerMedia: MediaQueryList | null = null
let reducedMotionMedia: MediaQueryList | null = null
let lockSettled = false
let isMounted = false
let activationVersion = 0

function readGsapNumber(element: HTMLElement, property: 'x' | 'y' | 'rotation'): number {
  if (!gsap) {
    return 0
  }
  return Number(gsap.getProperty(element, property)) || 0
}

function toRingCoordinates(point: Point): Point {
  const cursorElement = cursor.value
  const ringElement = ring.value
  if (!cursorElement || !ringElement) {
    return point
  }

  const radians = -readGsapNumber(ringElement, 'rotation') * Math.PI / 180
  const x = point.x - readGsapNumber(cursorElement, 'x')
  const y = point.y - readGsapNumber(cursorElement, 'y')
  return {
    x: x * Math.cos(radians) - y * Math.sin(radians),
    y: x * Math.sin(radians) + y * Math.cos(radians),
  }
}

function targetCornerPositions(target: HTMLElement): Record<CornerName, Point> {
  const bounds = target.getBoundingClientRect()
  return {
    'top-left': { x: bounds.left - CORNER_GAP, y: bounds.top - CORNER_GAP },
    'top-right': { x: bounds.right + CORNER_GAP, y: bounds.top - CORNER_GAP },
    'bottom-right': { x: bounds.right + CORNER_GAP, y: bounds.bottom + CORNER_GAP },
    'bottom-left': { x: bounds.left - CORNER_GAP, y: bounds.bottom + CORNER_GAP },
  }
}

function namedCornerElements(): Record<CornerName, HTMLElement> | null {
  const elements = cornerElements.value
  if (!elements) {
    return null
  }

  const entries = elements.flatMap((element) => {
    const name = element.dataset.cursorCorner
    return CORNERS.some(corner => corner.name === name)
      ? [[name as CornerName, element] as const]
      : []
  })
  if (entries.length !== CORNERS.length) {
    return null
  }
  return Object.fromEntries(entries) as Record<CornerName, HTMLElement>
}

function setCornerPositions(positions: Record<CornerName, Point>): void {
  const corners = namedCornerElements()
  if (!gsap || !corners) {
    return
  }
  CORNERS.forEach(({ name }) => gsap?.set(corners[name], positions[name]))
}

function updateLockedCorners(animate = false): void {
  const corners = namedCornerElements()
  if (!gsap || !lockedTarget || !corners) {
    return
  }

  const targetPositions = targetCornerPositions(lockedTarget)
  const positions = Object.fromEntries(
    CORNERS.map(({ name }) => [name, toRingCoordinates(targetPositions[name])]),
  ) as Record<CornerName, Point>
  if (!animate) {
    setCornerPositions(positions)
    return
  }

  lockSettled = false
  cornerTimeline?.kill()
  cornerTimeline = gsap.timeline({
    onComplete: () => {
      lockSettled = true
      updateLockedCorners()
    },
  })
  CORNERS.forEach(({ name }) => {
    cornerTimeline?.to(corners[name], {
      ...positions[name],
      duration: 0.2,
      ease: 'power2.out',
      overwrite: true,
    }, 0)
  })
}

function observeTarget(target: HTMLElement): void {
  targetResizeObserver?.disconnect()
  targetResizeObserver = new ResizeObserver(() => updateLockedCorners(!lockSettled))
  targetResizeObserver.observe(target)
}

function lock(target: HTMLElement): void {
  if (!gsap || lockedTarget === target) {
    return
  }

  lockedTarget = target
  locked.value = true
  spinTween?.pause()
  observeTarget(target)
  updateLockedCorners(true)
}

function unlock(): void {
  if (!gsap || !lockedTarget) {
    return
  }

  lockedTarget = null
  locked.value = false
  lockSettled = false
  targetResizeObserver?.disconnect()
  cornerTimeline?.kill()
  const corners = namedCornerElements()
  if (corners) {
    cornerTimeline = gsap.timeline()
    CORNERS.forEach(({ name }) => {
      cornerTimeline?.to(corners[name], {
        ...FREE_POSITIONS[name],
        duration: 0.24,
        ease: 'power3.out',
        overwrite: true,
      }, 0)
    })
  }
  spinTween?.play()
}

function onPointerMove(event: PointerEvent): void {
  moveCursorX?.(event.clientX)
  moveCursorY?.(event.clientY)
}

function onPointerOver(event: PointerEvent): void {
  const target = (event.target as Element | null)?.closest<HTMLElement>('.cursor-target')
  if (target) {
    lock(target)
  }
}

function onPointerOut(event: PointerEvent): void {
  const nextTarget = (event.relatedTarget as Element | null)?.closest<HTMLElement>('.cursor-target')
  if (nextTarget) {
    lock(nextTarget)
    return
  }
  unlock()
}

function onPointerDown(): void {
  if (gsap && dot.value) {
    gsap.to(dot.value, { scale: 0.65, duration: 0.12, ease: 'power2.out' })
  }
}

function onPointerUp(): void {
  if (gsap && dot.value) {
    gsap.to(dot.value, { scale: 1, duration: 0.16, ease: 'power2.out' })
  }
}

function onGeometryChange(): void {
  if (lockedTarget) {
    updateLockedCorners(!lockSettled)
  }
}

function trackLockedGeometry(): void {
  if (lockedTarget && lockSettled) {
    updateLockedCorners()
  }
}

function addListeners(): void {
  window.addEventListener('pointermove', onPointerMove, { passive: true })
  document.addEventListener('pointerover', onPointerOver)
  document.addEventListener('pointerout', onPointerOut)
  window.addEventListener('pointerdown', onPointerDown)
  window.addEventListener('pointerup', onPointerUp)
  window.addEventListener('pointercancel', onPointerUp)
  window.addEventListener('blur', onPointerUp)
  window.addEventListener('scroll', onGeometryChange, { passive: true })
  window.addEventListener('resize', onGeometryChange)
  gsap?.ticker.add(trackLockedGeometry)
}

function removeListeners(): void {
  window.removeEventListener('pointermove', onPointerMove)
  document.removeEventListener('pointerover', onPointerOver)
  document.removeEventListener('pointerout', onPointerOut)
  window.removeEventListener('pointerdown', onPointerDown)
  window.removeEventListener('pointerup', onPointerUp)
  window.removeEventListener('pointercancel', onPointerUp)
  window.removeEventListener('blur', onPointerUp)
  window.removeEventListener('scroll', onGeometryChange)
  window.removeEventListener('resize', onGeometryChange)
  gsap?.ticker.remove(trackLockedGeometry)
}

function shouldEnableCursor(): boolean {
  return Boolean(finePointerMedia?.matches && !reducedMotionMedia?.matches)
}

function disableCursor(): void {
  removeListeners()
  targetResizeObserver?.disconnect()
  targetResizeObserver = null
  cornerTimeline?.kill()
  cornerTimeline = null
  spinTween?.kill()
  spinTween = null
  const corners = namedCornerElements()
  if (gsap && corners) {
    gsap.killTweensOf(Object.values(corners))
  }
  if (gsap && dot.value) {
    gsap.killTweensOf(dot.value)
  }
  if (gsap && cursor.value) {
    gsap.killTweensOf(cursor.value)
  }
  moveCursorX = null
  moveCursorY = null
  lockedTarget = null
  lockSettled = false
  locked.value = false
  enabled.value = false
  document.documentElement.classList.remove('kagari-target-cursor')
}

async function reconcileCursor(): Promise<void> {
  const version = ++activationVersion
  if (!shouldEnableCursor()) {
    disableCursor()
    return
  }
  if (enabled.value) {
    return
  }

  const module = await import('gsap')
  if (!isMounted || version !== activationVersion || !shouldEnableCursor()) {
    return
  }
  gsap = module.gsap
  enabled.value = true
  await nextTick()

  const cursorElement = cursor.value
  const ringElement = ring.value
  const corners = namedCornerElements()
  if (!isMounted || version !== activationVersion || !shouldEnableCursor()) {
    return
  }
  if (!cursorElement || !ringElement || !corners) {
    disableCursor()
    return
  }

  const initialX = window.innerWidth / 2
  const initialY = window.innerHeight / 2
  gsap.set(cursorElement, { x: initialX, y: initialY })
  setCornerPositions(FREE_POSITIONS)
  moveCursorX = gsap.quickTo(cursorElement, 'x', { duration: 0.1, ease: 'power3.out' })
  moveCursorY = gsap.quickTo(cursorElement, 'y', { duration: 0.1, ease: 'power3.out' })
  spinTween = gsap.to(ringElement, {
    rotation: '+=360',
    duration: 2,
    ease: 'none',
    repeat: -1,
  })

  document.documentElement.classList.add('kagari-target-cursor')
  addListeners()
}

function onMediaConditionChange(): void {
  void reconcileCursor()
}

onMounted(() => {
  isMounted = true
  finePointerMedia = window.matchMedia('(hover: hover) and (pointer: fine)')
  reducedMotionMedia = window.matchMedia('(prefers-reduced-motion: reduce)')
  finePointerMedia.addEventListener('change', onMediaConditionChange)
  reducedMotionMedia.addEventListener('change', onMediaConditionChange)
  void reconcileCursor()
})

onBeforeUnmount(() => {
  isMounted = false
  activationVersion += 1
  finePointerMedia?.removeEventListener('change', onMediaConditionChange)
  reducedMotionMedia?.removeEventListener('change', onMediaConditionChange)
  finePointerMedia = null
  reducedMotionMedia = null
  disableCursor()
})
</script>

<template>
  <div
    v-if="enabled"
    ref="cursor"
    class="target-cursor"
    :class="{ 'target-cursor--locked': locked }"
    :data-locked="locked"
    data-testid="target-cursor"
    aria-hidden="true"
  >
    <span ref="dot" class="target-cursor__dot" />
    <span ref="ring" class="target-cursor__ring">
      <i
        v-for="cornerItem in CORNERS"
        :key="cornerItem.name"
        ref="corner"
        class="target-cursor__corner"
        :data-cursor-corner="cornerItem.name"
      >
        <span
          class="target-cursor__corner-mark"
          :class="cornerItem.markClass"
        />
      </i>
    </span>
  </div>
</template>

<style scoped>
:global(html.kagari-target-cursor),
:global(html.kagari-target-cursor *) {
  cursor: none !important;
}

.target-cursor {
  position: fixed;
  z-index: 110;
  top: 0;
  left: 0;
  width: 1px;
  height: 1px;
  pointer-events: none;
  color: rgba(236, 233, 223, 0.92);
  will-change: transform;
}

.target-cursor__ring,
.target-cursor__corner {
  position: absolute;
  top: 0;
  left: 0;
  width: 0;
  height: 0;
  will-change: transform;
}

.target-cursor__ring {
  transform-origin: 0 0;
}

.target-cursor__dot {
  position: absolute;
  top: -2px;
  left: -2px;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 8px rgba(236, 233, 223, 0.42);
  will-change: transform;
}

.target-cursor__corner-mark {
  position: absolute;
  display: block;
  width: 12px;
  height: 12px;
  border-color: currentColor;
  border-style: solid;
  transition: color 160ms ease;
}

.target-cursor__corner-mark--top-left {
  top: 0;
  left: 0;
  border-width: 2px 0 0 2px;
}

.target-cursor__corner-mark--top-right {
  top: 0;
  left: 0;
  transform: translateX(-100%);
  border-width: 2px 2px 0 0;
}

.target-cursor__corner-mark--bottom-right {
  top: 0;
  left: 0;
  transform: translate(-100%, -100%);
  border-width: 0 2px 2px 0;
}

.target-cursor__corner-mark--bottom-left {
  top: 0;
  left: 0;
  transform: translateY(-100%);
  border-width: 0 0 2px 2px;
}

.target-cursor--locked {
  color: rgba(180, 151, 207, 0.96);
}
</style>
