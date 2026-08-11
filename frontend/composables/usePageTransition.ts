export type PageTransitionPhase = 'idle' | 'covering' | 'covered' | 'revealing'

let coverResolver: (() => void) | null = null

export function usePageTransition() {
  const phase = useState<PageTransitionPhase>('page-transition-phase', () => 'idle')
  const sequence = useState('page-transition-sequence', () => 0)

  function cover(): Promise<void> {
    if (phase.value === 'covered') {
      return Promise.resolve()
    }
    if (phase.value === 'covering') {
      return new Promise((resolve) => {
        const previousResolver = coverResolver
        coverResolver = () => {
          previousResolver?.()
          resolve()
        }
      })
    }

    phase.value = 'covering'
    sequence.value += 1
    return new Promise((resolve) => {
      coverResolver = resolve
    })
  }

  function markCovered(): void {
    if (phase.value !== 'covering') {
      return
    }
    phase.value = 'covered'
    coverResolver?.()
    coverResolver = null
  }

  function reveal(): void {
    if (phase.value === 'idle') {
      return
    }
    phase.value = 'revealing'
  }

  function finish(): void {
    if (phase.value === 'revealing') {
      phase.value = 'idle'
    }
  }

  return {
    phase,
    sequence,
    cover,
    markCovered,
    reveal,
    finish,
  }
}
