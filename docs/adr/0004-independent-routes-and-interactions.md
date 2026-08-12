# ADR-0004: Independent Public Module Routes and Interaction Model

- Status: Accepted
- Date: 2026-08-10
- Superseded in part by: ADR-0005

## Context

The Archive Monolith Visual Prototype validated the selected visual language, but it placed multiple modules on one page solely to compare composition. The intended production information architecture requires dedicated, shareable public routes. Prototype feedback also identified the required behavior for the persistent Canvas background, desktop target cursor, and Album Item exploration.

## Decision

Use independent English-language public routes for home, works, blog, music, GitHub, gallery, service status, and visitor messages. The home page contains only curated summaries and entry points; every module page begins with a shared archive-style header before its unique content.

Render the global Canvas as a continuously drawn stateful grid. Each newly entered grid cell triggers a randomized 3×3 activation only once; activated cells retain a timestamp and fade in path order over approximately 2.5 seconds.

On desktop, use a GSAP-managed target cursor: it rotates while free, stops and locks four corners to a designated target on hover, then returns to free rotation when leaving. Do not display an action label.

Use a finite set of Album Items in a two-dimensional pointer-drag canvas. When items cross a world edge, reposition them on the opposite edge to create seamless four-direction wrapping. Do not add wheel movement, inertia, or a lightbox in the first release.

## Consequences

- Public information is independently addressable, indexable, and shareable.
- Navigation and home summaries must avoid duplicating full module content.
- Canvas and target cursor require client-only lifecycle management with frame-based rendering and cleanup.
- Gallery interaction must arbitrate touch scrolling only while an in-canvas drag is active.
- The prototype must be updated or replaced before implementation because its current Canvas, cursor, and gallery behavior are not production baselines.
