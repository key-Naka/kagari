<script setup lang="ts">
const { phase, sequence, finish, markCovered } = usePageTransition()

function onTransitionEnd(event: TransitionEvent): void {
  if (event.propertyName !== 'transform') {
    return
  }
  if (phase.value === 'covering') {
    markCovered()
    return
  }
  finish()
}
</script>

<template>
  <div
    class="page-transition-overlay"
    :class="`page-transition-overlay--${phase}`"
    :data-phase="phase"
    :data-sequence="sequence"
    data-testid="page-transition"
    aria-hidden="true"
    @transitionend="onTransitionEnd"
  >
    <div class="page-transition-overlay__mark">K</div>
    <span class="page-transition-overlay__line" />
  </div>
</template>

<style scoped>
.page-transition-overlay {
  position: fixed;
  z-index: 120;
  inset: 0;
  display: grid;
  place-items: center;
  overflow: hidden;
  color: #ece9df;
  background: #0a0a0c;
  transform: translateY(101%);
  opacity: 0;
  pointer-events: none;
  transition: transform 420ms cubic-bezier(0.76, 0, 0.24, 1), opacity 120ms linear;
}

.page-transition-overlay--covering,
.page-transition-overlay--covered {
  transform: translateY(0);
  opacity: 1;
  pointer-events: auto;
}

.page-transition-overlay--revealing {
  transform: translateY(-101%);
  opacity: 1;
  pointer-events: auto;
}

.page-transition-overlay__mark {
  display: grid;
  width: 3.5rem;
  height: 3.5rem;
  place-items: center;
  border: 1px solid rgba(224, 220, 210, 0.56);
  border-radius: 999px;
  font-family: Georgia, serif;
  font-size: 1.75rem;
  box-shadow: 0 0 2.5rem rgba(177, 155, 232, 0.22);
}

.page-transition-overlay__line {
  position: absolute;
  width: min(72vw, 52rem);
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(224, 220, 210, 0.7), transparent);
  transform: translateY(5rem);
}

@media (prefers-reduced-motion: reduce) {
  .page-transition-overlay {
    transition-duration: 1ms;
  }
}
</style>
