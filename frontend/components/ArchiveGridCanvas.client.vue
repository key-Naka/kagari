<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, shallowRef } from 'vue'

interface ActiveCell {
  strength: number
  touchedAt: number
}

type RenderState = 'idle' | 'running' | 'static'

const canvas = shallowRef<HTMLCanvasElement | null>(null)
const activationCount = shallowRef(0)
const activeCount = shallowRef(0)
const renderState = shallowRef<RenderState>('idle')

const cellSize = 56
const activationRadius = 140
const holdTime = 400
const fadeDuration = 800
const maximumOpacity = 0.72
const activeCells = new Map<number, ActiveCell>()

let animationFrame = 0
let width = 0
let height = 0
let columns = 0
let rows = 0
let context: CanvasRenderingContext2D | null = null
let reducedMotion = false
let mounted = false

function smoothFalloff(value: number): number {
  return value * value * (3 - 2 * value)
}

function resizeCanvas(): void {
  const element = canvas.value
  if (!element) {
    return
  }

  const ratio = Math.min(window.devicePixelRatio || 1, 2)
  width = window.innerWidth
  height = window.innerHeight
  columns = Math.ceil(width / cellSize)
  rows = Math.ceil(height / cellSize)
  element.width = Math.max(1, Math.round(width * ratio))
  element.height = Math.max(1, Math.round(height * ratio))
  element.style.width = `${width}px`
  element.style.height = `${height}px`
  context = element.getContext('2d')
  context?.setTransform(ratio, 0, 0, ratio, 0, 0)
}

function drawStaticGrid(): void {
  if (!context) {
    return
  }

  context.strokeStyle = 'rgba(214, 207, 193, 0.075)'
  context.lineWidth = 1
  context.beginPath()
  for (let x = 0; x <= width; x += cellSize) {
    context.moveTo(x + 0.5, 0)
    context.lineTo(x + 0.5, height)
  }
  for (let y = 0; y <= height; y += cellSize) {
    context.moveTo(0, y + 0.5)
    context.lineTo(width, y + 0.5)
  }
  context.stroke()
}

function drawActiveCell(index: number, opacity: number): void {
  if (!context) {
    return
  }

  const column = index % columns
  const row = Math.floor(index / columns)
  const x = column * cellSize + 1
  const y = row * cellSize + 1
  const size = cellSize - 2
  const centerX = x + size / 2
  const centerY = y + size / 2
  const gradient = context.createRadialGradient(
    centerX,
    centerY,
    cellSize * 0.08,
    centerX,
    centerY,
    cellSize * 0.82,
  )
  gradient.addColorStop(0, `rgba(179, 157, 235, ${opacity})`)
  gradient.addColorStop(1, 'rgba(179, 157, 235, 0)')

  context.strokeStyle = gradient
  context.lineWidth = 1.2
  context.strokeRect(x + 0.5, y + 0.5, size - 1, size - 1)
}

function drawFrame(timestamp: number): void {
  animationFrame = 0
  if (!mounted || reducedMotion || !context) {
    return
  }

  context.clearRect(0, 0, width, height)
  drawStaticGrid()

  for (const [index, cell] of activeCells) {
    const fadeElapsed = timestamp - cell.touchedAt - holdTime
    if (fadeElapsed >= fadeDuration) {
      activeCells.delete(index)
      continue
    }

    const opacity = fadeElapsed <= 0
      ? cell.strength
      : cell.strength * (1 - fadeElapsed / fadeDuration)
    drawActiveCell(index, opacity)
  }

  activeCount.value = activeCells.size
  if (activeCells.size > 0) {
    animationFrame = requestAnimationFrame(drawFrame)
    return
  }

  renderState.value = 'idle'
}

function wakeRenderer(): void {
  if (reducedMotion || animationFrame !== 0) {
    return
  }

  renderState.value = 'running'
  animationFrame = requestAnimationFrame(drawFrame)
}

function energizeCells(pointerX: number, pointerY: number): void {
  const now = performance.now()
  const minimumColumn = Math.max(0, Math.floor((pointerX - activationRadius) / cellSize))
  const maximumColumn = Math.min(columns - 1, Math.floor((pointerX + activationRadius) / cellSize))
  const minimumRow = Math.max(0, Math.floor((pointerY - activationRadius) / cellSize))
  const maximumRow = Math.min(rows - 1, Math.floor((pointerY + activationRadius) / cellSize))
  let energized = false

  for (let row = minimumRow; row <= maximumRow; row += 1) {
    for (let column = minimumColumn; column <= maximumColumn; column += 1) {
      const centerX = column * cellSize + cellSize / 2
      const centerY = row * cellSize + cellSize / 2
      const distance = Math.hypot(centerX - pointerX, centerY - pointerY)
      if (distance > activationRadius) {
        continue
      }

      const index = row * columns + column
      const strength = smoothFalloff(1 - distance / activationRadius) * maximumOpacity
      const current = activeCells.get(index)
      activeCells.set(index, {
        strength: Math.max(current?.strength ?? 0, strength),
        touchedAt: now,
      })
      energized = true
    }
  }

  if (!energized) {
    return
  }

  activationCount.value += 1
  activeCount.value = activeCells.size
  wakeRenderer()
}

function onPointerMove(event: PointerEvent): void {
  energizeCells(event.clientX, event.clientY)
}

function onResize(): void {
  activeCells.clear()
  activeCount.value = 0
  resizeCanvas()
  context?.clearRect(0, 0, width, height)
  drawStaticGrid()

  if (!reducedMotion) {
    wakeRenderer()
  }
}

onMounted(async () => {
  mounted = true
  reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  await nextTick()
  if (!mounted || !canvas.value) {
    return
  }
  resizeCanvas()
  drawStaticGrid()
  window.addEventListener('resize', onResize, { passive: true })

  if (reducedMotion) {
    renderState.value = 'static'
    return
  }

  renderState.value = 'idle'
  window.addEventListener('pointermove', onPointerMove, { passive: true })
})

onBeforeUnmount(() => {
  mounted = false
  cancelAnimationFrame(animationFrame)
  animationFrame = 0
  activeCells.clear()
  window.removeEventListener('resize', onResize)
  window.removeEventListener('pointermove', onPointerMove)
})
</script>

<template>
  <canvas
    ref="canvas"
    class="archive-grid-canvas"
    :data-activation-count="activationCount"
    :data-active-count="activeCount"
    :data-render-state="renderState"
    data-testid="archive-grid-canvas"
    aria-hidden="true"
  />
</template>

<style scoped>
.archive-grid-canvas {
  position: fixed;
  z-index: 30;
  inset: 0;
  display: block;
  pointer-events: none;
  mix-blend-mode: screen;
}

@media (prefers-reduced-motion: reduce) {
  .archive-grid-canvas {
    opacity: 0.2;
  }
}
</style>
