<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const cursor = ref<HTMLElement | null>(null)
const ring = ref<HTMLElement | null>(null)
const enabled = ref(false)
const locked = ref(false)
const pointerX = ref(0)
const pointerY = ref(0)
const targetBounds = ref<DOMRect | null>(null)
let lockedTarget: HTMLElement | null = null
let rotateTween: { play: () => void; pause: () => void; kill: () => void } | null = null
let isMounted = false

const cursorStyle = computed(() => {
  if (locked.value && targetBounds.value) {
    const bounds = targetBounds.value
    return {
      left: `${bounds.left}px`,
      top: `${bounds.top}px`,
      width: `${bounds.width}px`,
      height: `${bounds.height}px`,
    }
  }
  return {
    left: `${pointerX.value}px`,
    top: `${pointerY.value}px`,
    width: '0px',
    height: '0px',
  }
})

function updateBounds(): void {
  if (lockedTarget) {
    targetBounds.value = lockedTarget.getBoundingClientRect()
  }
}

function unlock(): void {
  locked.value = false
  lockedTarget = null
  targetBounds.value = null
  rotateTween?.play()
}

function lock(target: HTMLElement): void {
  lockedTarget = target
  updateBounds()
  locked.value = true
  rotateTween?.pause()
}

function onPointerMove(event: PointerEvent): void {
  pointerX.value = event.clientX
  pointerY.value = event.clientY
  updateBounds()
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

onMounted(async () => {
  const media = window.matchMedia('(hover: hover) and (pointer: fine)')
  enabled.value = media.matches
  if (!enabled.value) {
    return
  }
  isMounted = true
  pointerX.value = window.innerWidth / 2
  pointerY.value = window.innerHeight / 2
  window.addEventListener('pointermove', onPointerMove, { passive: true })
  document.addEventListener('pointerover', onPointerOver)
  document.addEventListener('pointerout', onPointerOut)
  window.addEventListener('scroll', updateBounds, { passive: true })
  window.addEventListener('resize', updateBounds)
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    return
  }
  const { gsap } = await import('gsap')
  if (isMounted && ring.value) {
    rotateTween = gsap.to(ring.value, {
      rotation: 360,
      duration: 5,
      ease: 'none',
      repeat: -1,
    })
  }
})

onBeforeUnmount(() => {
  isMounted = false
  rotateTween?.kill()
  window.removeEventListener('pointermove', onPointerMove)
  document.removeEventListener('pointerover', onPointerOver)
  document.removeEventListener('pointerout', onPointerOut)
  window.removeEventListener('scroll', updateBounds)
  window.removeEventListener('resize', updateBounds)
})
</script>

<template>
  <div
    v-if="enabled"
    ref="cursor"
    class="target-cursor"
    :class="{ 'target-cursor--locked': locked }"
    :style="cursorStyle"
    :data-locked="locked"
    data-testid="target-cursor"
    aria-hidden="true"
  >
    <span ref="ring" class="target-cursor__ring">
      <i class="target-cursor__corner target-cursor__corner--top-left" />
      <i class="target-cursor__corner target-cursor__corner--top-right" />
      <i class="target-cursor__corner target-cursor__corner--bottom-right" />
      <i class="target-cursor__corner target-cursor__corner--bottom-left" />
    </span>
  </div>
</template>

<style scoped>
.target-cursor {
  position: fixed;
  z-index: 110;
  pointer-events: none;
  transform: translate(-50%, -50%);
  transition: left 100ms ease-out, top 100ms ease-out, width 160ms ease-out, height 160ms ease-out;
}

.target-cursor--locked {
  transform: none;
}

.target-cursor__ring {
  position: absolute;
  inset: 0;
}

.target-cursor__corner {
  position: absolute;
  width: 1rem;
  height: 1rem;
  border-color: rgba(236, 233, 223, 0.9);
  border-style: solid;
  transition: width 160ms ease-out, height 160ms ease-out, border-color 160ms ease-out;
}

.target-cursor__corner--top-left {
  top: -1rem;
  left: -1rem;
  border-width: 1px 0 0 1px;
}

.target-cursor__corner--top-right {
  top: -1rem;
  right: -1rem;
  border-width: 1px 1px 0 0;
}

.target-cursor__corner--bottom-right {
  right: -1rem;
  bottom: -1rem;
  border-width: 0 1px 1px 0;
}

.target-cursor__corner--bottom-left {
  bottom: -1rem;
  left: -1rem;
  border-width: 0 0 1px 1px;
}

.target-cursor--locked .target-cursor__corner {
  width: 1.35rem;
  height: 1.35rem;
  border-color: rgba(221, 192, 128, 0.95);
}

@media (prefers-reduced-motion: reduce) {
  .target-cursor,
  .target-cursor__corner {
    transition-duration: 1ms;
  }
}
</style>
