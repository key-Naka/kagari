# Kagari Personal Site

A single-author public portfolio and blog where the owner curates creative work, writing, media, and public service information. Visitors browse public content and can leave messages without an account.

## Content

**Portfolio Project**:
A curated work item represented by cover images, descriptive content, a public website link, and a source or repository address.
_Avoid_: Case study, product

**Blog Post**:
An owner-authored Markdown article managed in the administration console and grouped into time-based archives.
_Avoid_: Note, page

**Archive**:
A year-and-month grouping of published Blog Posts for chronological browsing.
_Avoid_: Deleted post, backup

**Album Item**:
An image managed by the owner and positioned as part of an infinite two-dimensional draggable canvas gallery with seamless edge wrapping.
_Avoid_: Attachment, asset, masonry gallery

**Track**:
A locally hosted audio work available to play in the public music module, with owner-curated metadata and an automatically detected duration.
_Avoid_: Playlist item, stream

## Community

**Visitor Message**:
A public message submitted by an anonymous visitor or a visitor using a nickname; the owner can remove it from the administration console.
_Avoid_: Comment, review

## Operations

**Service Status**:
The public presentation of sanitized host, container, and application metrics; raw infrastructure identifiers remain private.
_Avoid_: Health check, uptime

**Contribution Heatmap**:
A year-long visual summary of the owner's public GitHub contribution activity.
_Avoid_: Activity chart, private contribution graph

**Rate Limit**:
A Redis-backed boundary that limits visitor message submissions by source IP and route over a short time window.
_Avoid_: Moderation, CAPTCHA

**Visual Prototype**:
The first implementation stage that validates the site's visual language, navigation, Canvas background, transition/loading states, and public module composition before production integrations.
_Avoid_: MVP, production release

**Mini Player**:
A persistent compact music control that remains visible after a visitor leaves the independent music module while playback continues.
_Avoid_: Global autoplay, audio widget

**Obsidian Sigil Navigation**:
The site's dark floating pill navigation: translucent obsidian surfaces, restrained silver outlines, liquid hover fill, and a floating mark for the active route.
_Avoid_: Generic pill nav, white capsule menu

**Ritual Mobile Menu**:
A full-screen navigation overlay that uses the site's transition language to reveal and dismiss all public routes on mobile and the desktop More index.
_Avoid_: Hamburger dropdown

**Administration Console**:
The owner-only management interface for publishing and managing the site's content, visitor messages, and configuration.
_Avoid_: Dashboard, CMS
