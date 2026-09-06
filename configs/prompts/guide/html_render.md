## HTML Render Guide

One HTML file — report, dashboard, chart, diagram, map. No build step, no local assets, no server.

### 1. Check the gallery — always, before writing a tag

`http_request` https://view.agenvoy.com/list — JSON, one entry per page with `name`, `category`, `title`, `desc`, `url`.

Match on **shape**, not subject: what the reader *does* with the page — skim a status, compare options, follow a flow, click through a mock, answer questions before merging. The `unknowns/*` entries are the ones built to surface what the reader has not decided yet.

- Something fits → **§2**. "Close enough" fits.
- Nothing fits → **§3**, after naming the closest two entries and why each falls short.
- No HTTP possible, or the call failed → **§3**, nothing to announce.

### 2. An example fits — copy it, do not design

1. Fetch its `url` as raw DOM: `fetch_page(type="html")` or `http_request`. Never the markdown default — it strips the class names you are about to reuse.
2. Copy its `<link rel="stylesheet">` tag into your page.
3. Replace the content in place, keeping its class names and block order.
4. Save with `edit_file`.

**Zero CSS.** No `<style>`, no `style=`, no custom property, no font stack. Redefining one variable — `--accent`, `--ink` — repaints the whole page.

**Zero new class names.** Content that fits no component goes into the nearest one that exists: a stat block, a card, a list, a callout. Inventing `.profile-grid` or `.repo-list` is what forces a stylesheet to be written. If nothing can hold the content, the wrong entry was picked — go back to §1.

**Zero sample data.** The brand and every figure in the example are fictional.

**Zero deliberation.** Typography, palette, spacing and breakpoints arrived with the sheet. §3 and below do not apply on this branch.

The rest is boilerplate: doctype, charset, viewport meta, `<title>`, that one `<link>`, any CDN library the content uses, body markup.

### 3. Nothing fits — build it

Everything below applies only here. The stylesheet and a small `T` runtime are yours to write into the file.

- **The document is yours end to end** — `<html>`, `<head>`, viewport meta, a real `<title>`, one inline `<style>`, one inline `<script>`.
- **Charts** go through `T.init(elementId, option)`; give the container a height (`<div class="plot" id="gex" style="aspect-ratio:2/1">`).
- **Colours** come from `T`, never a hex literal — `T.series[0..3]`, `T.up`/`T.down`, `T.good`/`T.critical`, `T.ink`/`T.muted`/`T.rule`/`T.surface`, `T.tone(n)`; numbers through `T.fmt`/`T.signed`/`T.si`; tables through `T.sortable(...)`.
- **Layout** comes from the house classes — `.masthead`, `.band`, `.stats/.stat`, `.exhibit`, `.plot`, `.note`, `.tw`, `.cols`, `.pill`, `.flag`, `ul.pts`, `.grid/.card`.
- **Data** goes in one `const DATA = {...}` at the top of the script.
- **Chart everything chartable**; tables are for data that resists plotting. Cite sources as `<a href>` using a URL that appeared in tool output, never a constructed one.
- **Format is not a question to ask.** Produce text unless a page, file, link, html or pdf was asked for; when it was, write the file and return its path instead of also pasting the report back.

#### Libraries

Load one only when the content uses it. Six exist, nothing else — a chart ECharts cannot draw is a chart to reconsider.

| Need | Load | CDN |
|---|---|---|
| Charts | ECharts 5 | `https://cdn.jsdelivr.net/npm/echarts@5/dist/echarts.min.js` |
| Icons | Font Awesome 6 | `https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@6/css/all.min.css` |
| Code blocks | highlight.js 11 | `https://cdn.jsdelivr.net/npm/@highlightjs/cdn-assets@11/highlight.min.js` + `styles/github.min.css` |
| Maths | KaTeX 0.16 | `https://cdn.jsdelivr.net/npm/katex@0.16/dist/katex.min.js` + `katex.min.css` + `contrib/auto-render.min.js` |
| Flow / sequence / Gantt | Mermaid 11 | `https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs` (ESM) |
| Maps | MapLibre GL 4 | `https://cdn.jsdelivr.net/npm/maplibre-gl@4/dist/maplibre-gl.js` + `maplibre-gl.css` |

- Major-only paths are safe; never invent a minor version.
- **CSS matters as much as JS** — MapLibre without its sheet is a blank box, KaTeX without its sheet shows raw boxes.
- **KaTeX** needs both scripts, then one `renderMathInElement(document.body, {delimiters:[...]})`.
- **Mermaid** ships 3.5 MB as UMD against 30 KB as ESM — import the ESM entry, `mermaid.initialize({startOnLoad:false})`, then `await mermaid.run()`.
- highlight.js and KaTeX initialise **after** the content exists — at the end of the body.

#### Responsive

Mobile-first. Base styles are the phone; each query widens it.

```css
/* base — under 480: single column, stacked */
@media (min-width: 480px)  { /* two-up stat tiles, larger type scale */ }
@media (min-width: 640px)  { /* side-by-side content, persistent nav */ }
@media (min-width: 1024px) { /* multi-column grid, fixed max-width */ }
```

- **The page never scrolls horizontally.** Wide tables, code blocks and timelines scroll inside their own `overflow-x: auto` container.
- **Charts resize with their container** — ECharts measures once, so call `chart.resize()` from a `ResizeObserver`. The SVG renderer prints better than canvas.
- **MapLibre needs `map.resize()`** or it stretches.
- **Every chart, map and canvas needs a height that works at 480px** — a percentage height in an auto-height parent collapses to zero.
- `img`, `svg`, `canvas`, `iframe` carry `max-width: 100%`. Use `clamp()` for type and spacing.
- **Anything interactive keeps a visible keyboard focus ring** — sortable headers, links, controls.

#### Visual direction

Decide before writing markup and state the direction in one line as you build. Without it every page lands on the same default — system sans, blue accent, rounded white cards on grey — which reads as unconsidered however correct the data is. A brief that pins an axis wins on that axis, its own words exactly; the axes it leaves open are yours to choose on.

Name the subject, its reader, and the page's one job before choosing anything. Distinctive choices come out of that subject's own world — its instruments, artifacts and vernacular — rather than out of a palette that would fit any dataset.

- **Typography** — a display face and a body face chosen as a pair, with a scale whose weights and widths are deliberate. The type treatment is part of what makes the page memorable, not a neutral delivery vehicle for the numbers.
- **Palette** — 4–6 named values: one accent, a neutral ramp, semantic colours kept apart from the accent. Financial, scientific, editorial and monitoring content do not share a palette. They are written once into `T` and read back as `T.*`, so no hex literal appears in the markup.
- **Spacing** — one rhythm (4px or 8px base) applied to padding, gaps and vertical spacing alike.
- **Background** — flat white is a default, not a decision. Consider a tinted surface or a dark canvas.
- **Motion** — one orchestrated moment (a load sequence, a scroll reveal) lands harder than micro-interactions scattered across the page, and the scattered version is what reads as machine-generated. Honour `prefers-reduced-motion`.
- **Signature** — the one element the page is remembered by. Spend boldness there and keep everything around it quiet.

Three looks arrive by default rather than by choice, and they turn up whatever the subject is: cream ground near `#F4F1EA` with a high-contrast serif display and a terracotta accent; near-black ground with a single acid-green or vermilion accent; broadsheet with hairline rules, zero border-radius and dense newspaper columns. Each is right when the brief asks for it — on a free axis, spend the freedom somewhere else.

Structure carries information: numbering, eyebrows, dividers and labels either encode something true about the content or they decorate it. `01 / 02 / 03` fits a real sequence and misleads on a set of peers.

Match execution to the direction — maximalist needs elaborate follow-through, minimal needs precision in spacing, type and detail.

Labels name what the reader recognises rather than what the pipeline produced, and a panel with no data says what is missing and what would fill it.

#### Plan, critique, build

Write the token block before any markup: palette values, type roles (display / body / utility), the layout concept in one sentence, the signature element. Read it back against the brief — a part that would come out identical for any other page on this subject is a default, so change it and name what changed. The code then follows the revised plan, every colour and type decision traceable to it.

Keep selector specificity flat as you write: a class rule and an element rule covering the same box (`.section` against `section`) cancel each other, and it surfaces as vertical spacing that silently goes missing between blocks.
