## HTML Render Guide

Contract for producing a standalone HTML deliverable — report, dashboard, chart, map, or 3D view. Everything ships in one file: no build step, no local assets, no server.

### Library selection

Load a library only when the content actually uses it. An unused `<script>` costs load time and buys nothing; a dashboard of static numbers needs none of these.

| The content needs | Load | CDN |
|---|---|---|
| Data-driven charts, scales, axes, custom SVG visualisation | d3 v7 | `https://cdn.jsdelivr.net/npm/d3@7/dist/d3.min.js` |
| 3D scenes, WebGL, geometry | three.js | `https://cdn.jsdelivr.net/npm/three@<pin>/build/three.module.js` |
| Interactive maps, geo overlays, tiles | MapLibre GL JS v4 | `https://cdn.jsdelivr.net/npm/maplibre-gl@4/dist/maplibre-gl.js` + matching `maplibre-gl.css` |
| Icons | Font Awesome 6 | `https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@6/css/all.min.css` |

Version notes — read before pasting:

- Verify the exact version resolves before shipping; do not invent a minor version. Major-only paths (`d3@7`, `maplibre-gl@4`) are safe.
- three.js addons (`OrbitControls`, loaders) live under a **version-pinned** path and break on a floating major. Pin one explicit version and use an import map:
  ```html
  <script type="importmap">
  {"imports":{"three":"https://cdn.jsdelivr.net/npm/three@<pin>/build/three.module.js",
              "three/addons/":"https://cdn.jsdelivr.net/npm/three@<pin>/examples/jsm/"}}
  </script>
  <script type="module">import * as THREE from 'three';</script>
  ```
- MapLibre needs its stylesheet as well as its script; without the CSS the map renders as a blank box with no controls.
- d3 and MapLibre coexist fine — d3 for the chart layer, MapLibre for the basemap.

### Responsive layout

Mobile-first with three breakpoints. Base styles are the phone layout; each query widens it.

```css
/* base — under 480: single column, stacked, full-bleed cards */
@media (min-width: 480px)  { /* large phone: two-up stat tiles, larger type scale */ }
@media (min-width: 800px)  { /* tablet / small laptop: side-by-side content, persistent nav */ }
@media (min-width: 1024px) { /* desktop: full multi-column grid, fixed max-width container */ }
```

Rules that hold at every width:

- **The page never scrolls horizontally.** Wide content — tables, code blocks, timelines — scrolls inside its own `overflow-x: auto` container instead of pushing the body.
- **Charts resize with their container.** For d3, render into an SVG with `viewBox` and `preserveAspectRatio` so it scales for free, or redraw from a `ResizeObserver` when the layout genuinely changes (axis tick density, label rotation).
- **Canvas-backed views need an explicit resize handler.** three.js: update `camera.aspect`, call `camera.updateProjectionMatrix()`, then `renderer.setSize(...)`. MapLibre: call `map.resize()`. Neither reflows on its own, and both stretch visibly if skipped.
- **Give every chart, map, and canvas container a height that works at 480px** — a percentage height inside an auto-height parent collapses to zero and renders nothing.
- `img`, `svg`, `canvas`, `iframe` carry `max-width: 100%`.
- Use `clamp()` for type and spacing so the three breakpoints handle layout changes rather than font-size churn.

### Visual direction

Decide the look before writing markup, and state the four choices — typography, palette, spacing rhythm, background — in one line to the user as you build. A page assembled without those decisions lands on the same default every time: system sans, a blue accent, evenly rounded white cards on grey. That default reads as unconsidered regardless of how correct the data is.

- **Typography** — pick a specific pairing and a scale. Vary weight and size with intent; a page where every heading is the same bold step has no hierarchy.
- **Palette** — one accent plus a neutral ramp, chosen for the subject. Financial, scientific, editorial, and monitoring content do not share a palette. Semantic colours (gain/loss, severity) are separate from the accent and must survive on both light and dark surfaces.
- **Spacing** — commit to one rhythm (a 4px or 8px base) and apply it to padding, gaps, and vertical spacing alike. Inconsistent gaps read as sloppiness faster than any colour choice.
- **Background** — flat white is a decision made by default, not on purpose. Consider a tinted surface, a subtle gradient, or a dark canvas when the content suits it.
- **Motion** — only where it carries meaning: chart entry, state transition, hover affordance. Decorative animation on a data page is noise. Honour `prefers-reduced-motion`.

When the user has stated a direction, follow it. When they have not and the subject admits several plausible looks, name the one you are taking rather than defaulting silently.

### Structure

- One `<style>` block and one `<script>` block, both inline. Keep datasets in a single `const DATA = {...}` at the top of the script, not scattered through render calls.
- Set `<meta name="viewport" content="width=device-width, initial-scale=1">` — without it the breakpoints never fire on a phone.
- Give the page a real `<title>`; it names the browser tab and the PDF if the user prints it.
