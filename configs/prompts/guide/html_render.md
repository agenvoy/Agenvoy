## HTML Render Guide

Contract for producing an HTML deliverable — report, dashboard, chart, diagram, or map.

### Which path

**A hosted renderer is in the tool list** (`generate_view` and anything shaped like it: takes `body_html` / `script` / `data`, returns a URL) → use it. It owns the page shell, so the rules are its rules, not this file's:

- **Only the libraries that tool names exist**, and the renderer injects their CDN tags itself — write no `<script>`/`<link>` of your own. `generate_view` carries six, each auto-detected from your markup and script (`libs` only forces one in):

  | Need | Library | Detected from |
  |---|---|---|
  | Charts | ECharts 5 | `T.init(` / `echarts.` |
  | Icons | Font Awesome 6 | `fa-` classes |
  | Code blocks | highlight.js 11 | `<pre><code` / `class="language-…"` |
  | Maths | KaTeX 0.16 | `$$…$$` / `\(…\)` / `renderMathInElement` |
  | Flow / sequence / Gantt diagrams | Mermaid 11 | `<div class="mermaid">` |
  | Maps | MapLibre GL 4 | `maplibregl` |

  highlight.js, KaTeX and Mermaid are initialised for you after your script runs — write the markup, not the bootstrap. Nothing outside those six is available; the table further down applies to the self-contained path only.
- **No boilerplate.** The house stylesheet and its `T` runtime are injected; author content only — no `<html>`, `<head>`, `<style>` scaffolding, no viewport meta.
- **Charts go through the runtime**: `T.init(elementId, option)` applies the theme, renderer and sizing. Give the container a height (`<div class="plot" id="gex" style="aspect-ratio:2/1">`).
- **Colours come from `T`**, never a hex literal — `T.series[0..3]`, `T.up`/`T.down`, `T.good`/`T.critical`, `T.ink`/`T.muted`/`T.rule`/`T.surface`, `T.tone(n)` for sign colouring; `T.fmt`/`T.signed`/`T.si` for numbers; `T.sortable(...)` for sortable tables.
- **Datasets go in `data`** as a JSON string (read back as `window.DATA`), not inlined into `script`.
- **Layout comes from its classes** (`.masthead`, `.band`, `.stats/.stat`, `.exhibit`, `.plot`, `.note`, `.tw`, `.cols`, `.pill`, `.flag`, `ul.pts`, `.grid/.card`) rather than hand-rolled CSS.
- **Chart everything chartable**; tables are for data that resists plotting. Cite every headline, post or filing as `<a href>` using a URL that appeared in tool output — never a constructed one.
- **Format is not a question to ask.** Produce text unless the user asked for a page, a file, a link, or said html/pdf; when they did, render and return the URL rather than also pasting the whole report back.

**No such tool available** → the self-contained single file described below: one HTML file, no build step, no local assets, no server.

### Library selection

Load a library only when the content actually uses it. An unused `<script>` costs load time and buys nothing; a dashboard of static numbers needs none of these.

The same six libraries as the hosted path, so a page can move between the two without a rewrite. Nothing else — a chart that ECharts cannot draw is a chart to reconsider, not a reason to add a second charting library.

| The content needs | Load | CDN |
|---|---|---|
| Any chart, distribution, ranking, breakdown | ECharts 5 | `https://cdn.jsdelivr.net/npm/echarts@5/dist/echarts.min.js` |
| Icons | Font Awesome 6 | `https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@6/css/all.min.css` |
| Code blocks | highlight.js 11 | `https://cdn.jsdelivr.net/npm/@highlightjs/cdn-assets@11/highlight.min.js` + `styles/github.min.css` |
| Maths | KaTeX 0.16 | `https://cdn.jsdelivr.net/npm/katex@0.16/dist/katex.min.js` + `katex.min.css` + `contrib/auto-render.min.js` |
| Flow, sequence, Gantt diagrams | Mermaid 11 | `https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs` (ESM) |
| Interactive maps, geo overlays, tiles | MapLibre GL 4 | `https://cdn.jsdelivr.net/npm/maplibre-gl@4/dist/maplibre-gl.js` + matching `maplibre-gl.css` |

Notes — read before pasting:

- Major-only paths (`echarts@5`, `maplibre-gl@4`) are safe; never invent a minor version.
- **CSS matters as much as JS.** MapLibre without its stylesheet renders a blank box with no controls; highlight.js without a theme leaves code unstyled; KaTeX without its CSS shows raw boxes.
- **KaTeX needs two scripts** — the engine and `auto-render`, then one `renderMathInElement(document.body, {delimiters:[…]})` call.
- **Mermaid ships a 3.5 MB UMD bundle against a 30 KB ESM entry** — import the ESM one, `mermaid.initialize({startOnLoad:false})`, then `await mermaid.run()`.
- highlight.js and KaTeX both need their init call **after** the content exists; run them at the end of the body, not inline beside the markup.

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
- **Charts resize with their container.** ECharts needs `chart.resize()` from a `ResizeObserver` or the `resize` event — it measures once at init and never reflows on its own. The SVG renderer (`echarts.init(el,null,{renderer:"svg"})`) scales more cleanly for print and PDF than canvas.
- **Map views need an explicit resize handler.** MapLibre: call `map.resize()`; it does not reflow on its own and stretches visibly if skipped.
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
