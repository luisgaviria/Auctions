---
name: frontend
description: 'Use when implementing, debugging, reviewing, or testing this repository’s Astro frontend: pages, SSR/API data loading, TypeScript browser behavior, Svelte/MapLibre maps, styling, SEO, and frontend-to-Go API integration.'
---

# Frontend Development

Use this skill for changes under `frontend/` and for work that changes the browser-facing contract with the Go API.

## Repository Map

- `src/pages/` defines Astro routes, including the auction index, Massachusetts location pages, favorites/auth pages, and property reports.
- `src/layouts/Layout.astro` provides shared document metadata, top navigation, fonts, theme setup, and Astro `ClientRouter` navigation.
- `src/components/` contains Astro presentation components; `Map.svelte` owns the interactive MapLibre map.
- `src/scripts/` contains browser-side TypeScript behavior, including shared state, search, pagination, favorites, map view, and drawer interactions.
- `src/styles/` contains component/page styles. Prefer the stylesheet already paired with the component or page being changed.

## Working Procedure

1. Trace the route and data flow before editing: page-level API request, component props/markup, browser script or Svelte component, and its styles. Check the matching Go response shape when changing API data usage.
2. Keep page rendering in Astro where practical and use client-side code only for interactions. The map is mounted lazily; preserve that behavior so MapLibre does not load for users who stay in grid view.
3. The auction index serializes initial SSR data and configuration into `#page-config` data attributes; `src/scripts/state.ts` hydrates shared state from that element. Keep attribute names, serialization, and client parsing in sync when changing this contract.
4. Map/list coordination uses DOM `CustomEvent`s such as `markerclick` and `auctionresultschange`. Preserve event names and detail shapes across the Svelte map and browser scripts, or update both sides together.
5. Account for `ClientRouter`: browser setup tied only to `DOMContentLoaded` runs on the initial document load, not automatically after every client-side navigation. Use the appropriate Astro lifecycle event or make initialization explicitly safe to rerun, and clean up listeners where needed.
6. Keep API data handling resilient to absent/null values. Backend auction fields such as dates, deposits, and coordinates are serialized strings; check `models.AuctionJSON` when assumptions about their shape change.
7. Preserve semantic HTML, accessible labels/keyboard behavior, responsive layouts, and the existing visual system. Keep page/component styles scoped to the feature and avoid broad formatting churn.
8. Run the frontend build after changes and review the final diff for contract, navigation, and responsive regressions.

## Validation

From `frontend/`:

```sh
npm run build
```

This runs `astro check` and the production build. For interactive changes, also run `npm run dev` and verify the affected flow in a browser at desktop and mobile widths. The package currently has no dedicated test script. `npm run format` formats the entire frontend tree; avoid using it for a narrowly scoped change unless that broad rewrite is intended.

## Configuration Notes

- `VITE_API_URL` configures the frontend API base URL. Its current fallback is `http://localhost:8080`; the Go backend defaults to port `8000`, so configure the two consistently for local development.
- Map style setup reads `VITE_MAPTILER_KEY` in `Map.svelte`. `VITE_*` values are bundled for the browser and must not contain private credentials.
- Treat rendered auction data as externally sourced content. Escape or safely render values before inserting them into HTML, especially in map popups; do not interpolate untrusted strings into raw HTML without an established sanitization strategy.
