# Auction Intel — auction-site-astro

A full-stack Massachusetts real estate auction aggregator. The Go backend scrapes 12 auctioneer websites daily and stores results in Supabase. The Astro 6 frontend displays them in a minimal "financial terminal" UI.

---

## Stack

| Layer | Technology |
|---|---|
| Frontend | Astro 6, React, Satoshi + JetBrains Mono (Fonts API) |
| Backend | Go 1.22, gorilla/mux, goquery, colly |
| Database | PostgreSQL via Supabase |
| Scraping | Cloudflare Browser Rendering API + Gemini AI fallback |
| Auth | JWT (HS256) |
| CI | GitHub Actions |

---

## Project Phases

### Phase 7 — Card Redesign
Removed card images entirely. Switched to a typography-focused 4-row layout: status pill → street address → city/state → terminal data block (Date / Time / Deposit). Added JSON-LD `Event` schema markup per card for SEO. Responsive 3→2→1 column grid.

### Phase 8 — Astro 6 Upgrade & Industrial Redesign
Upgraded from Astro 4.15 → 6.0.8 and `@astrojs/react` 3.6 → 5.0.1. Replaced `<ViewTransitions />` with `<ClientRouter />` for SPA routing. Implemented the Astro 6 Fonts API with `fontProviders.fontshare()` (Satoshi) and `fontProviders.fontsource()` (JetBrains Mono). Built the "Financial Terminal" design system: Zinc neutral palette, Deep Jade accent (`hsl(160, 75%, 18%)`), 2px border-radius cards, AUCTION.INTEL wordmark nav.

### Phase 8 Extended — High-End Industrial Redesign
Overhauled the design system to a Zinc-based neutral palette with WCAG AA contrast on every text element. Replaced the floating pill nav with a full-width bar. Introduced HSL CSS variables with light/dark tokens for every colour role. Verified all contrast ratios: zinc-600 (#52525b) = 7:1 on white; zinc-400 (#a1a1aa) = 7:1 on zinc-900.

### Phase 8 Navigation — Topbar Overhaul
Rewrote `Topbar.astro` to use `<style is:global>` (fixes Astro CSS scoping — JS-injected auth links were invisible in dark mode because Astro scoped styles don't apply to `innerHTML`-created elements). Nav is fully transparent with a frosted glass `backdrop-filter: blur(12px)` effect and a 72% opacity tint that adapts to both themes. Active pill uses `background: var(--color-text-primary); color: var(--color-bg)` — auto-inverts correctly in both light and dark.

### Phase 9 — Typography Hardening & Accessibility
Hardened terminal block legibility. Labels switched to `var(--color-accent)` (jade green / neon mint) for high contrast and visual hierarchy. Values bumped to `font-weight: 600` and `0.8rem`. Added a 2px accent-colored top border on the terminal block as a visual anchor. Removed the grey background approach in favour of colour-based hierarchy. Confirmed no `font-weight: 100/200` overrides anywhere in the stylesheet.

### Phase 10 — Data Integrity & Pagination Recovery
Four fixes in one phase:
1. **Patriot deposit extraction** — scraper now searches `.auction-terms` first, falls back to the old nth-child selector, and extracts the first `$X,XXX` dollar amount via regex instead of brittle string splitting.
2. **Time sanitization** — `normalizeTime()` added to `utils/scrap.go`. Nulls out `00:00:00`, `00:00`, `12:00 AM` (default midnight values scrapers emit when no real time is known) before the DB write.
3. **ClientRouter / Load More fix** — replaced `DOMContentLoaded` with `astro:page-load` in `index.astro` and `favorites.astro`. With Astro 6's `<ClientRouter />`, `DOMContentLoaded` only fires on the initial hard load; `astro:page-load` fires on every SPA navigation, so Load More and favorites persist across page transitions.
4. **Time TBD fallback** — empty time fields render as italic muted "TBD" via `.terminal-value--tbd` (zinc-500, weight 400, italic) in Card.astro, index.astro, and favorites.astro.

### Phase 12 — Environment-Aware CORS Architecture
Replaced the single `FRONTEND_URL` CORS config with a dynamic multi-origin system. `config.GetAllowedOrigins()` reads `ALLOWED_ORIGINS` (comma-separated), falls back to `FRONTEND_URL`, then to `localhost:4321` with a warning log. The middleware builds an O(1) map at startup and only sets `Access-Control-Allow-Origin` when the request origin is in the allowed set. Local `.env` includes ports 4321, 4322, and 3000.

### Phase 13 — GitHub Actions Autopilot
Created `.github/workflows/scrape.yml`. Runs all 12 scrapers daily at 12:00 UTC (8:00 AM EST) via `cmd/dryrun`. Also supports `workflow_dispatch` for manual trigger from the GitHub Actions tab. Uses `ENV: PROD` to skip `godotenv` on the runner. Go module cache is keyed to `backend/go.sum` to speed up repeat runs.

**Required GitHub Secrets:**

| Secret | Purpose |
|---|---|
| `DATABASE_URL` | Supabase PostgreSQL connection string |
| `CF_ACCOUNT_ID` | Cloudflare Browser Rendering account ID |
| `CF_API_TOKEN` | Cloudflare Browser Rendering API token |
| `GEMINI_API_KEY` | Gemini AI fallback extractor key |

---

## Local Development

### Backend

```bash
cd backend
cp .env.example .env   # fill in your values
go run ./cmd/dryrun    # test scrape
go run .               # start API server on :8080
```

### Frontend

```bash
cd frontend
npm install
npm run dev            # starts on localhost:4321
```

### Environment variables (backend `.env`)

```
DATABASE_URL=postgresql://...
CF_ACCOUNT_ID=...
CF_API_TOKEN=...
GEMINI_API_KEY=...
JWT_SECRET=...
PORT=8080
ENV=development
ALLOWED_ORIGINS=http://localhost:4321,http://localhost:4322,http://localhost:3000
```

---

## Architecture

```
browser
  └── Astro 6 SSR (frontend :4321)
        ├── Card.astro          terminal layout, JSON-LD schema
        ├── Topbar.astro        frosted glass nav, JWT auth state
        └── pages/
              ├── index.astro   SSR initial 20 auctions + Load More
              └── favorites.astro  JWT-gated saved auctions

Go API (:8080)
  ├── GET  /auctions?limit=&offset=
  ├── POST /favorites/add|remove
  ├── GET  /favorites
  └── POST /scraping/start  (async, 15-min timeout)

Scrapers (12 parallel goroutines)
  ├── CF Browser Rendering → baystate, commonwealth, patriot
  ├── Legacy colly/goquery → amg, apg, danielp, dean, harvard, jake, sri, sullivan, tache
  └── Gemini AI fallback   → triggered when scraper returns 0 or >50% empty addresses

Supabase (PostgreSQL)
  └── auctions table  ON CONFLICT (address, site_name) DO UPDATE
```

---

## References

- Go password auth: https://www.sohamkamani.com/golang/password-authentication-and-storage/
- Go session/cookie auth: https://www.sohamkamani.com/golang/session-cookie-authentication/
