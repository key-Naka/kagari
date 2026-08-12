# ADR-0005: Responsive Grid, Target Cursor, and Complete Navigation

- Status: Accepted
- Date: 2026-08-12
- Supersedes in part: ADR-0003 and ADR-0004

## Context

Production review of the global interaction system found that permanently visited Canvas cells stopped responding, the target cursor expanded as a fixed box instead of preserving its rotating four-corner motion, non-home navigation labels could disappear when the liquid fill did not run, and the desktop More index caused transparent content conflicts.

## Decision

Replace the one-time randomized 3x3 Canvas activation with a repeatable cursor-radius field. Use 56px cells, a 140px smooth falloff radius, a 400ms hold, an 800ms fade, a maximum opacity of 0.72, purple gradient borders without active fill, and the existing faint silver lattice. Run animation frames only while cells are active or rendering has been explicitly awakened. Do not add click pulses. With reduced motion enabled, render only the static lattice.

On fine-pointer desktop devices, hide the native cursor and use a GSAP-managed target cursor with four independently positioned corners and a center dot. The free cursor follows with a short ease and rotates every two seconds. On target entry, stop the free rotation, reset it to zero degrees, and animate each orthogonal corner to three pixels outside the target's measured bounds; continue measuring during scrolling and layout changes. On exit, return the corners to the free cursor and restart rotation from zero degrees. Pointer press scales both the center dot and the complete cursor. Keep `.cursor-target` as the explicit target boundary. Do not render the custom cursor under reduced motion or on touch/mobile conditions.

Desktop navigation directly exposes all eight public routes: 首页、作品、博客、音乐、相册、GitHub、服务状态 and 访客留言. Remove the desktop More trigger. Keep the Ritual Mobile Menu below 1120px and on touch-oriented layouts. Liquid hover fill applies only to inactive routes and must have a CSS fallback so labels remain readable before GSAP loads and under reduced motion. Active routes retain their purple fill and active sigil.

## Consequences

- Canvas cells can respond again after fading instead of becoming permanently inert.
- The interaction loop sleeps while idle, reducing background rendering work.
- Cursor motion preserves the rotating four-corner visual language while accurately following target geometry.
- Every desktop destination is directly reachable without a transparent overlay.
- Browser coverage must verify repeat activation, full-page pointer response, native-cursor hiding, independent corner locking, eight visible desktop routes, the 1120px breakpoint, reduced-motion behavior, and horizontal overflow.
