# ADR-0003: Obsidian Sigil Navigation

- Status: Accepted
- Date: 2026-08-10

## Context

The public site has multiple routes and needs a recognizable navigation treatment that matches its dark gothic-fantasy visual direction. The referenced PillNav interaction provides a compact pill layout, liquid hover fill, active-route marker, and a mobile menu pattern, but its original light capsule styling does not match the site's visual language.

## Decision

Use an Obsidian Sigil Navigation: a floating translucent dark pill navigation with restrained silver borders, GSAP-driven liquid hover fills, and a small luminous sigil beneath the active route. Desktop navigation directly shows 首页、作品、博客、音乐、相册、GitHub and opens a full-screen index for the remaining routes. The index is also the mobile navigation, exposes every public route, uses Chinese labels, and opens with the Ritual Mobile Menu's transition language. The navigation begins floating and tightens after 32px of vertical scroll.

## Consequences

- Navigation remains visually distinctive without competing with portfolio content.
- GSAP becomes an approved dependency for navigation and other high-value timeline interactions.
- Route labels and active state must remain accessible to keyboard and screen-reader users.
- The full-screen index locks page scrolling, dims the underlying content and Canvas, traps keyboard focus, closes with Escape or an overlay click, and restores focus to its trigger.
- A route chosen from the full-screen index first closes the menu, then begins the global page-transition overlay.
- The K sigil performs a brief rotation on hover and emits a short glow on navigation.
