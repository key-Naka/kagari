<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

interface NavigationItem {
  label: string
  to: string
}

const route = useRoute()
const compact = ref(false)
const menuOpen = ref(false)
const pendingRoute = ref<string | null>(null)
const menu = ref<HTMLElement | null>(null)
const closeButton = ref<HTMLButtonElement | null>(null)
const menuTrigger = ref<HTMLElement | null>(null)
const sigil = ref<HTMLElement | null>(null)
let bodyOverflow = ''
let rotateSigil: ((target: HTMLElement) => void) | null = null
let isMounted = false

const publicRoutes: NavigationItem[] = [
  { label: '首页', to: '/' },
  { label: '作品', to: '/works' },
  { label: '博客', to: '/blog' },
  { label: '音乐', to: '/music' },
  { label: '相册', to: '/gallery' },
  { label: 'GitHub', to: '/github' },
  { label: '服务状态', to: '/status' },
  { label: '访客留言', to: '/visitor-messages' },
]

const currentRoute = computed(() => route.path)

function isActive(item: NavigationItem): boolean {
  return item.to === '/'
    ? currentRoute.value === '/'
    : currentRoute.value === item.to || currentRoute.value.startsWith(`${item.to}/`)
}

function updateCompactNavigation(): void {
  compact.value = window.scrollY > 32
}

function openMenu(event: MouseEvent): void {
  menuTrigger.value = event.currentTarget instanceof HTMLElement ? event.currentTarget : null
  pendingRoute.value = null
  menuOpen.value = true
}

function closeMenu(): void {
  menuOpen.value = false
}

function chooseMenuRoute(to: string): void {
  pendingRoute.value = to
  menuOpen.value = false
}

function onMenuTransitionEnd(event: TransitionEvent): void {
  if (event.target !== menu.value || event.propertyName !== 'opacity' || menuOpen.value) {
    return
  }
  const destination = pendingRoute.value
  pendingRoute.value = null
  if (destination) {
    void navigateTo(destination)
    return
  }
  menuTrigger.value?.focus()
}

function focusableMenuItems(): HTMLElement[] {
  return menu.value
    ? Array.from(menu.value.querySelectorAll<HTMLElement>('a[href], button:not([disabled])'))
    : []
}

function onMenuKeydown(event: KeyboardEvent): void {
  if (!menuOpen.value) {
    return
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    closeMenu()
    return
  }
  if (event.key !== 'Tab') {
    return
  }
  const items = focusableMenuItems()
  if (items.length === 0) {
    event.preventDefault()
    return
  }
  const first = items[0]
  const last = items[items.length - 1]
  if (!first || !last) {
    return
  }
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

function glowSigil(): void {
  if (sigil.value) {
    rotateSigil?.(sigil.value)
  }
}

watch(menuOpen, async (isOpen) => {
  if (isOpen) {
    bodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    await nextTick()
    closeButton.value?.focus()
    return
  }
  document.body.style.overflow = bodyOverflow
})

onMounted(async () => {
  isMounted = true
  updateCompactNavigation()
  window.addEventListener('scroll', updateCompactNavigation, { passive: true })
  window.addEventListener('keydown', onMenuKeydown)
  window.addEventListener('kagari:navigation', glowSigil)

  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    return
  }
  const { gsap } = await import('gsap')
  if (!isMounted) {
    return
  }
  rotateSigil = (target) => {
    gsap.fromTo(target, {
      boxShadow: '0 0 0 rgba(179, 157, 235, 0)',
      rotation: 0,
    }, {
      boxShadow: '0 0 28px rgba(179, 157, 235, 0.72)',
      duration: 0.18,
      rotation: 180,
      yoyo: true,
      repeat: 1,
      ease: 'power2.out',
    })
  }
})

onBeforeUnmount(() => {
  isMounted = false
  window.removeEventListener('scroll', updateCompactNavigation)
  window.removeEventListener('keydown', onMenuKeydown)
  window.removeEventListener('kagari:navigation', glowSigil)
  document.body.style.overflow = bodyOverflow
})
</script>

<template>
  <header
    class="site-navigation"
    :class="{ 'site-navigation--compact': compact }"
    data-testid="site-navigation"
  >
    <NuxtLink
      to="/"
      class="site-navigation__sigil cursor-target"
      aria-label="返回首页"
      title="返回首页"
      @mouseenter="glowSigil"
      @click="glowSigil"
    >
      <span ref="sigil">K</span>
    </NuxtLink>

    <nav class="site-navigation__routes" aria-label="主导航">
      <NuxtLink
        v-for="item in publicRoutes"
        :key="item.to"
        :to="item.to"
        class="site-navigation__route cursor-target"
        :class="{ 'site-navigation__route--active': isActive(item) }"
        :aria-current="isActive(item) ? 'page' : undefined"
      >
        <span data-liquid-fill class="site-navigation__liquid" aria-hidden="true" />
        <span class="site-navigation__label">{{ item.label }}</span>
        <span v-if="isActive(item)" class="site-navigation__active-sigil" aria-hidden="true" />
      </NuxtLink>
    </nav>

    <button
      type="button"
      class="site-navigation__mobile-trigger cursor-target"
      aria-label="打开全部导航"
      :aria-expanded="menuOpen"
      aria-controls="ritual-menu"
      @click="openMenu"
    >
      <span />
      <span />
      <span />
    </button>

    <div
      id="ritual-menu"
      ref="menu"
      class="ritual-menu"
      :class="{ 'ritual-menu--open': menuOpen }"
      :data-open="menuOpen"
      data-testid="ritual-menu"
      aria-label="全部导航"
      role="dialog"
      :aria-hidden="!menuOpen"
      :aria-modal="menuOpen ? 'true' : undefined"
      :inert="!menuOpen"
      @click.self="closeMenu"
      @transitionend="onMenuTransitionEnd"
    >
      <button
        ref="closeButton"
        type="button"
        class="ritual-menu__close cursor-target"
        aria-label="关闭导航"
        title="关闭导航"
        @click="closeMenu"
      >
        <span aria-hidden="true">+</span>
      </button>
      <nav class="ritual-menu__routes" aria-label="全部公开路由">
        <NuxtLink
          v-for="item in publicRoutes"
          :key="item.to"
          :to="item.to"
          class="ritual-menu__route cursor-target"
          :class="{ 'ritual-menu__route--active': isActive(item) }"
          :aria-current="isActive(item) ? 'page' : undefined"
          @click.prevent="chooseMenuRoute(item.to)"
        >
          <span>{{ item.label }}</span>
          <small>{{ item.to }}</small>
        </NuxtLink>
      </nav>
    </div>
  </header>
</template>

<style scoped>
.site-navigation {
  position: fixed;
  z-index: 80;
  top: 1rem;
  left: 50%;
  display: flex;
  width: min(92vw, 72rem);
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  transform: translateX(-50%);
  transition: transform 260ms ease, top 260ms ease;
}

.site-navigation--compact {
  top: 0.5rem;
}

.site-navigation__sigil,
.site-navigation__mobile-trigger {
  display: grid;
  width: 2.75rem;
  height: 2.75rem;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid rgba(224, 220, 210, 0.35);
  border-radius: 999px;
  color: #f2efe7;
  background: rgba(9, 9, 11, 0.82);
  box-shadow: inset 0 0 1.1rem rgba(179, 157, 235, 0.11);
  text-decoration: none;
  backdrop-filter: blur(18px);
}

.site-navigation__sigil {
  font-family: Georgia, serif;
  font-size: 1.35rem;
  cursor: pointer;
}

.site-navigation__routes {
  display: flex;
  min-height: 2.75rem;
  align-items: stretch;
  gap: 0.125rem;
  padding: 0.2rem;
  border: 1px solid rgba(224, 220, 210, 0.3);
  border-radius: 999px;
  background: rgba(9, 9, 11, 0.82);
  backdrop-filter: blur(18px);
}

.site-navigation__route {
  position: relative;
  display: grid;
  min-width: 3.6rem;
  place-items: center;
  overflow: hidden;
  border: 0;
  border-radius: 999px;
  padding: 0 0.8rem;
  color: #b9b7b2;
  background: transparent;
  font-size: 0.75rem;
  text-decoration: none;
  transition: color 180ms ease;
  cursor: pointer;
}

.site-navigation__route:not(.site-navigation__route--active):hover,
.site-navigation__route:not(.site-navigation__route--active):focus-visible {
  color: #111113;
  outline: none;
}

.site-navigation__sigil:focus-visible,
.site-navigation__mobile-trigger:focus-visible,
.ritual-menu__close:focus-visible {
  outline: 2px solid #d9bf81;
  outline-offset: 3px;
}

.site-navigation__route--active {
  color: #f2efe7;
}

.site-navigation__label {
  position: relative;
  z-index: 1;
}

.site-navigation__liquid {
  position: absolute;
  z-index: 0;
  inset: 0;
  border-radius: inherit;
  background: #dedad1;
  transform: translateY(120%);
  transition: transform 340ms cubic-bezier(0.16, 1, 0.3, 1);
}

.site-navigation__route:not(.site-navigation__route--active):hover .site-navigation__liquid,
.site-navigation__route:not(.site-navigation__route--active):focus-visible .site-navigation__liquid {
  transform: translateY(0);
}

.site-navigation__route--active .site-navigation__liquid {
  background: rgba(179, 157, 235, 0.16);
  transform: translateY(0);
}

.site-navigation__route--active .site-navigation__label {
  color: #e9e4f6;
}

.site-navigation__active-sigil {
  position: absolute;
  z-index: 2;
  bottom: 0.15rem;
  left: 50%;
  width: 0.28rem;
  height: 0.28rem;
  border-radius: 999px;
  background: #b39deb;
  box-shadow: 0 0 0.75rem #b39deb;
  transform: translateX(-50%);
}

.site-navigation__mobile-trigger {
  display: none;
  grid-template-rows: repeat(3, 1px);
  align-content: center;
  gap: 0.25rem;
  border-radius: 999px;
  cursor: pointer;
}

.site-navigation__mobile-trigger span {
  width: 1rem;
  height: 1px;
  background: #e8e4db;
}

.ritual-menu {
  position: fixed;
  z-index: 90;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 5rem max(1.5rem, 6vw);
  color: #f2efe7;
  background: rgba(5, 5, 7, 0.96);
  opacity: 0;
  pointer-events: none;
  clip-path: circle(0 at 92% 8%);
  transition: opacity 360ms ease, clip-path 560ms cubic-bezier(0.16, 1, 0.3, 1);
}

.ritual-menu--open {
  opacity: 1;
  pointer-events: auto;
  clip-path: circle(150% at 92% 8%);
}

.ritual-menu__close {
  position: absolute;
  top: 1.25rem;
  right: max(1.25rem, 4vw);
  display: grid;
  width: 2.75rem;
  height: 2.75rem;
  place-items: center;
  border: 1px solid rgba(224, 220, 210, 0.35);
  border-radius: 999px;
  color: #f2efe7;
  background: transparent;
  font-size: 1.5rem;
  cursor: pointer;
}

.ritual-menu__close span {
  transform: rotate(45deg);
}

.ritual-menu__routes {
  display: grid;
  width: min(100%, 52rem);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem 3rem;
}

.ritual-menu__route {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid rgba(224, 220, 210, 0.24);
  padding: 1rem 0;
  color: #e7e3d9;
  font-family: Georgia, serif;
  font-size: clamp(2rem, 5vw, 4.5rem);
  line-height: 1;
  text-decoration: none;
  transition: color 180ms ease, padding 180ms ease;
}

.ritual-menu__route small {
  color: #797872;
  font-family: ui-monospace, monospace;
  font-size: 0.65rem;
}

.ritual-menu__route:hover,
.ritual-menu__route:focus-visible,
.ritual-menu__route--active {
  padding-left: 0.55rem;
  color: #d9bf81;
  outline: none;
}

.site-navigation__route:focus-visible,
.ritual-menu__route:focus-visible {
  box-shadow: 0 0 0 2px #d9bf81;
}

@media (max-width: 1119px), (hover: none), (pointer: coarse) {
  .site-navigation {
    width: calc(100% - 2rem);
  }

  .site-navigation__routes {
    display: none;
  }

  .site-navigation__mobile-trigger {
    display: grid;
  }

  .ritual-menu__routes {
    grid-template-columns: 1fr;
    gap: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .site-navigation,
  .ritual-menu,
  .site-navigation__route,
  .site-navigation__liquid,
  .ritual-menu__route {
    transition-duration: 1ms;
  }
}
</style>
