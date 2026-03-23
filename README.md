<div align="center">
  <img src="https://img.shields.io/badge/Astro_6-BC52EE?style=flat-square&logo=astro&logoColor=white" />
  <img src="https://img.shields.io/badge/Go_1.22-00ADD8?style=flat-square&logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/PostgreSQL-4169E1?style=flat-square&logo=postgresql&logoColor=white" />
  <img src="https://img.shields.io/badge/MapLibre-212121?style=flat-square&logo=maplibre&logoColor=white" />
</div>

<h1 align="center">Auction & Company</h1>
<p align="center">
  <strong>Massachusetts Real Estate Auction Aggregator</strong><br />
  Automated Daily Scraping • Spatial Mapping • AI Data Recovery
</p>

---

## 🏗️ Core Problem

Real estate auction data in Massachusetts is fragmented across 12+ legacy websites. These sites often lack search, mobile support, or structured data. This project automates the collection, cleaning, and mapping of that data into a single interface.

## 🛠️ Technical Stack

<table width="100%">
  <tr>
    <td width="50%" valign="top">
      <strong>Backend (Go)</strong>
      <ul>
        <li><b>Concurrency:</b> Parallel scraping via Goroutines/WaitGroups.</li>
        <li><b>Bot Bypass:</b> Cloudflare Browser Rendering API for JS-heavy sites.</li>
        <li><b>Self-Healing:</b> Gemini 2.5 Flash fallback for failed selectors.</li>
        <li><b>Automation:</b> GitHub Actions cron-jobs for daily updates.</li>
      </ul>
    </td>
    <td width="50%" valign="top">
      <strong>Frontend (Astro)</strong>
      <ul>
        <li><b>Architecture:</b> Astro 6 SSR for SEO and fast initial loads.</li>
        <li><b>Interactivity:</b> Svelte 5 for the Map and real-time filtering.</li>
        <li><b>Maps:</b> MapLibre GL with Bounding Box viewport queries.</li>
        <li><b>Design:</b> Industrial Zinc/Jade UI with JetBrains Mono.</li>
      </ul>
    </td>
  </tr>
</table>

---

## 🗺️ Spatial Engineering & Interaction

The map is a custom integration designed for high-density data.

- **Lazy Loading:** MapLibre (300KB JS) only hydrates when "Show Map" is clicked to maintain high Lighthouse scores.
- **Bi-Directional Sync:** \* Clicking a sidebar card triggers a `flyTo` transition on the map.
  - Clicking a map marker scrolls the sidebar to the matching card and triggers a highlight animation.
- **Viewport Filtering:** The backend queries Supabase using the map's current bounding box coordinates to filter results in real-time.

---

## 📈 Engineering Highlights

### Data Integrity & Cleanup

- **Automated Stale Handling:** The system marks auctions as "Removed" if they are no longer found during the daily scrape, ensuring the map stays current.
- **Regex Extraction:** Scrapers use regex to find deposit amounts ($X,XXX) within unstructured text blocks.
- **Time Normalization:** A cleaning layer converts inconsistent site times (e.g., "12:00 AM" or "Midnight") into null values or "TBD" for UI consistency.

### DevOps & CI/CD

- **Scrape Pipeline:** GitHub Actions runs a daily "Scrape → Geocode → Cleanup" cycle.
- **CORS Security:** A dynamic middleware builds an O(1) lookup table for allowed origins (Local, Preview, Production).

---

## 🚀 Local Setup

### 1. Backend

```bash
cd backend
go mod download
go run . # API starts on :8080
```
