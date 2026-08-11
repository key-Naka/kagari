<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

interface ActivatedCell {
  startedAt: number
  delay: number
  strength: number
}

const canvas = ref<HTMLCanvasElement | null>(null)
const activationCount = ref(0)
const activeCount = ref(0)
const cellSize = 44
const activationDuration = 2500
const maximumActivationDelay = 300
const activatedCells = new Map<string, ActivatedCell>()
const visitedCells = new Set<string>()
let animationFrame = 0
let expiryTimer: ReturnType<typeof setTimeout> | null = null
let lastCell = ''
let width = 0
let height = 0
let context: CanvasRenderingContext2D | null = null
let reducedMotion = false

function cellKey(column: number, row: number): string {
  return `${column}:${row}`
}

function resizeCanvas(): void {
  const element = canvas.value
  if (!element) {
    return
  }
  const ratio = Math.min(window.devicePixelRatio || 1, 1.5)
  width = window.innerWidth
  height = window.innerHeight
  element.width = Math.round(width * ratio)
  element.height = Math.round(height * ratio)
  element.style.width = `${width}px`
  element.style.height = `${height}px`
  context = element.getContext('2d')
  context?.setTransform(ratio, 0, 0, ratio, 0, 0)
}

function activateCell(column: number, row: number): void {
  const origin = cellKey(column, row)
  if (visitedCells.has(origin)) {
    return
  }
  visitedCells.add(origin)
  activationCount.value += 1
  const startedAt = performance.now()

  for (let offsetY = -1; offsetY <= 1; offsetY += 1) {
    for (let offsetX = -1; offsetX <= 1; offsetX += 1) {
      const isCenter = offsetX === 0 && offsetY === 0
      if (!isCenter && Math.random() < 0.38) {
        continue
      }
      const key = cellKey(column + offsetX, row + offsetY)
      activatedCells.set(key, {
        startedAt,
        delay: (Math.abs(offsetX) + Math.abs(offsetY)) * 105 + Math.random() * 90,
        strength: isCenter ? 1 : 0.44 + Math.random() * 0.36,
      })
    }
  }
  activeCount.value = activatedCells.size
  scheduleExpiredCellPrune()
}

function onPointerMove(event: PointerEvent): void {
  const column = Math.floor(event.clientX / cellSize)
  const row = Math.floor(event.clientY / cellSize)
  const key = cellKey(column, row)
  if (key === lastCell) {
    return
  }
  lastCell = key
  activateCell(column, row)
}

function pruneExpiredCells(timestamp: number): void {
  for (const [key, cell] of activatedCells) {
    if (timestamp - cell.startedAt - cell.delay >= activationDuration) {
      activatedCells.delete(key)
    }
  }
  activeCount.value = activatedCells.size
}

function scheduleExpiredCellPrune(): void {
  if (expiryTimer) {
    clearTimeout(expiryTimer)
  }
  expiryTimer = setTimeout(() => {
    expiryTimer = null
    pruneExpiredCells(performance.now())
  }, activationDuration + maximumActivationDelay)
}

function drawGrid(timestamp: number, scheduleNextFrame = true): void {
  if (!context) {
    if (scheduleNextFrame) {
      animationFrame = requestAnimationFrame(drawGrid)
    }
    return
  }

  context.clearRect(0, 0, width, height)
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

  pruneExpiredCells(timestamp)
  for (const [key, cell] of activatedCells) {
    const elapsed = timestamp - cell.startedAt - cell.delay
    if (elapsed < 0) {
      continue
    }
    const [column, row] = key.split(':').map(Number)
    if (column === undefined || row === undefined) {
      continue
    }
    const progress = elapsed / activationDuration
    const opacity = (1 - progress) * cell.strength * 0.3
    context.fillStyle = `rgba(179, 157, 235, ${opacity})`
    context.fillRect(column * cellSize + 2, row * cellSize + 2, cellSize - 4, cellSize - 4)
  }
  if (scheduleNextFrame) {
    animationFrame = requestAnimationFrame(drawGrid)
  }
}

function onResize(): void {
  resizeCanvas()
  if (reducedMotion) {
    drawGrid(performance.now(), false)
  }
}

onMounted(() => {
  reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  resizeCanvas()
  window.addEventListener('resize', onResize)
  if (reducedMotion) {
    drawGrid(performance.now(), false)
    return
  }
  window.addEventListener('pointermove', onPointerMove, { passive: true })
  animationFrame = requestAnimationFrame(drawGrid)
})

onBeforeUnmount(() => {
  cancelAnimationFrame(animationFrame)
  if (expiryTimer) {
    clearTimeout(expiryTimer)
  }
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
  opacity: 0.52;
  mix-blend-mode: screen;
}

@media (prefers-reduced-motion: reduce) {
  .archive-grid-canvas {
    opacity: 0.2;
  }
}
</style>
