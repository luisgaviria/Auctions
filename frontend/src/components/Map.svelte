<script>
  import { onMount, onDestroy } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';

  export let auctions = [];
  export let apiUrl = '';

  const MAPTILER_KEY = import.meta.env.VITE_MAPTILER_API_KEY;

  let mapContainer;
  let map = null;
  // Each entry: { id, marker, lngLat, popup }
  let markers = [];
  let activePopup = null;
  let hasMoved = false;
  let searching = false;
  let currentBounds = null;

  function formatSiteName(s) {
    if (!s) return '';
    return s.replace(/([a-z])([A-Z])/g, '$1 $2')
      .split(/[\s_-]+/)
      .map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
      .join(' ');
  }

  function validCoord(a) {
    const lat = parseFloat(a.lat), lng = parseFloat(a.lng);
    return !isNaN(lat) && !isNaN(lng) && lat !== 0 && lng !== 0;
  }

  function clearMarkers() {
    markers.forEach(({ marker }) => marker.remove());
    markers = [];
    if (activePopup) { activePopup.remove(); activePopup = null; }
  }

  function renderMarkers(auctionList) {
    if (!map) return;
    clearMarkers();

    auctionList.filter(validCoord).forEach(auction => {
      const lat = parseFloat(auction.lat);
      const lng = parseFloat(auction.lng);
      const lngLat = [lng, lat];

      const el = document.createElement('div');
      el.className = 'map-marker';

      const popup = new maplibregl.Popup({
        offset: 18,
        closeButton: false,
        maxWidth: '280px',
        className: 'auction-popup',
      }).setHTML(`
        <div class="popup-inner">
          <p class="popup-address">${auction.address || 'Address unavailable'}</p>
          ${auction.city && auction.state ? `<p class="popup-location">${auction.city}, ${auction.state}</p>` : ''}
          <div class="popup-meta">
            ${auction.date ? `<span class="popup-chip">${auction.date}</span>` : ''}
            ${auction.deposit ? `<span class="popup-chip">${auction.deposit}</span>` : ''}
          </div>
          ${auction.site_name ? `<p class="popup-source">${formatSiteName(auction.site_name)}</p>` : ''}
          ${auction.link ? `<a href="${auction.link}" target="_blank" rel="noopener noreferrer" class="popup-cta">View Details →</a>` : ''}
        </div>
      `);

      el.addEventListener('click', () => {
        if (activePopup) activePopup.remove();
        popup.setLngLat(lngLat).addTo(map);
        activePopup = popup;

        // Notify the sidebar so it can scroll to and highlight the matching card.
        mapContainer.dispatchEvent(new CustomEvent('markerclick', {
          bubbles: true,
          detail: { id: auction.id },
        }));
      });

      const marker = new maplibregl.Marker({ element: el, anchor: 'center' })
        .setLngLat(lngLat)
        .addTo(map);

      markers.push({ id: auction.id, marker, lngLat, popup });
    });
  }

  function onUserMoveEnd() {
    const b = map.getBounds();
    currentBounds = {
      north: b.getNorth(),
      south: b.getSouth(),
      east:  b.getEast(),
      west:  b.getWest(),
    };
    hasMoved = true;
  }

  async function searchArea() {
    if (!currentBounds || searching) return;
    searching = true;
    hasMoved = false;

    const { north, south, east, west } = currentBounds;
    const url = `${apiUrl}/auctions?north=${north}&south=${south}&east=${east}&west=${west}`;

    try {
      const res = await fetch(url);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      const newAuctions = data.auctions ?? [];

      renderMarkers(newAuctions);

      mapContainer.dispatchEvent(new CustomEvent('auctionresultschange', {
        bubbles: true,
        detail: { auctions: newAuctions },
      }));
    } catch (err) {
      console.error('[Map] bbox search failed:', err);
      hasMoved = true; // restore button so user can retry
    } finally {
      searching = false;
    }
  }

  onMount(() => {
    // Guard against HMR double-initialisation: onDestroy nulls map out,
    // so if this fires again before the old instance is torn down we skip.
    if (map) return;

    map = new maplibregl.Map({
      container: mapContainer,
      style: `https://api.maptiler.com/maps/streets-v2/style.json?key=${MAPTILER_KEY}`,
      center: [-71.0589, 42.3601],
      zoom: 9,
    });

    map.addControl(new maplibregl.NavigationControl(), 'top-right');

    map.on('load', () => {
      renderMarkers(auctions);

      const valid = auctions.filter(validCoord);
      let didFit = false;

      if (valid.length === 1) {
        map.flyTo({ center: [parseFloat(valid[0].lng), parseFloat(valid[0].lat)], zoom: 14 });
        didFit = true;
      } else if (valid.length > 1) {
        const bounds = new maplibregl.LngLatBounds();
        valid.forEach(a => bounds.extend([parseFloat(a.lng), parseFloat(a.lat)]));
        map.fitBounds(bounds, { padding: 60, maxZoom: 14 });
        didFit = true;
      }

      // After the initial fit animation settles, auto-search the visible viewport so
      // the sidebar is populated immediately without requiring the user to move the map.
      // Then register the user-movement tracker for all subsequent moves.
      function initialSearch() {
        const b = map.getBounds();
        currentBounds = {
          north: b.getNorth(),
          south: b.getSouth(),
          east:  b.getEast(),
          west:  b.getWest(),
        };
        searchArea();
        map.on('moveend', onUserMoveEnd);
      }

      if (didFit) {
        map.once('moveend', initialSearch);
      } else {
        initialSearch();
      }
    });

    map.on('click', (e) => {
      if (!e.originalEvent.target.closest('.map-marker') && activePopup) {
        activePopup.remove();
        activePopup = null;
      }
    });

    // 'resizemap' is dispatched by setView() in index.astro after the split panel
    // becomes visible.  MapLibre's canvas is 0×0 while the container is display:none,
    // so map.resize() must be called once the container has real dimensions.
    mapContainer.addEventListener('resizemap', () => {
      map.resize();
    });

    // 'flytomarker' is dispatched by the sidebar card click handler in index.astro.
    mapContainer.addEventListener('flytomarker', (e) => {
      const entry = markers.find(m => m.id === e.detail.id);
      if (!entry) return;

      // Close any open popup first.
      if (activePopup) { activePopup.remove(); activePopup = null; }

      // Fly to the marker, then open its popup once the animation settles.
      map.flyTo({ center: entry.lngLat, zoom: 15, duration: 700 });
      map.once('moveend', () => {
        entry.popup.setLngLat(entry.lngLat).addTo(map);
        activePopup = entry.popup;
      });
    });
  });

  onDestroy(() => {
    clearMarkers();
    if (map) {
      map.remove();
      map = null; // prevent HMR 'container already initialized' on re-mount
    }
  });
</script>

<div bind:this={mapContainer} class="map-container">
  {#if hasMoved || searching}
    <button
      class="search-area-btn"
      class:loading={searching}
      on:click={searchArea}
      disabled={searching}
      aria-label="Search auctions in this area"
    >
      {#if searching}
        <span class="btn-spinner" aria-hidden="true"></span>
        Searching…
      {:else}
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        Search this area
      {/if}
    </button>
  {/if}
</div>

<style>
  .map-container {
    position: relative;
    width: 100%;
    height: 100%;
    min-height: 480px;
    border-radius: 12px;
    overflow: hidden;
  }

  /* ── Search this area button ── */
  .search-area-btn {
    position: absolute;
    top: 1rem;
    left: 50%;
    transform: translateX(-50%);
    z-index: 10;

    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 0.5rem 1rem;
    border-radius: 999px;
    border: 1px solid rgba(228, 228, 231, 0.6);
    cursor: pointer;

    font-family: var(--font-sans, system-ui, sans-serif);
    font-size: 0.8125rem;
    font-weight: 600;
    letter-spacing: -0.01em;
    color: #09090b;

    background: rgba(255, 255, 255, 0.85);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.12), 0 1px 3px rgba(0, 0, 0, 0.08);
    transition: transform 0.15s ease, box-shadow 0.15s ease, opacity 0.15s ease;
    white-space: nowrap;
  }

  .search-area-btn:hover:not(:disabled) {
    transform: translateX(-50%) translateY(-1px);
    box-shadow: 0 4px 18px rgba(0, 0, 0, 0.16), 0 1px 4px rgba(0, 0, 0, 0.1);
  }

  .search-area-btn:disabled {
    opacity: 0.75;
    cursor: default;
  }

  /*
   * Dark mode: only the ancestor selector ([data-theme='dark']) is :global.
   * .search-area-btn and .btn-spinner keep their Svelte scoping hash so the
   * rule correctly targets the elements rendered in this component's template.
   */
  :global([data-theme='dark']) .search-area-btn {
    background: rgba(24, 24, 27, 0.85);
    border-color: rgba(63, 63, 70, 0.6);
    color: #fafafa;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.4);
  }

  :global([data-theme='dark']) .search-area-btn .btn-spinner {
    border-color: rgba(255, 255, 255, 0.2);
    border-top-color: #fafafa;
  }

  .btn-spinner {
    display: block;
    width: 11px;
    height: 11px;
    border: 1.5px solid rgba(0, 0, 0, 0.2);
    border-top-color: #09090b;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
    flex-shrink: 0;
  }

  /* spin is used only inside this component's scoped template — no -global- needed */
  @keyframes spin { to { transform: rotate(360deg); } }

  /*
   * .map-marker is created via document.createElement (not in Svelte's template),
   * so it never gets a scoping hash — :global() is required.
   *
   * @keyframes that are referenced from a :global() rule must also be global.
   * Svelte achieves this with the -global- prefix: the compiler strips it and
   * emits the keyframe name without any scope hash.
   */
  :global(.map-marker) {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--color-accent, #00b37e);
    border: 2px solid #fff;
    cursor: pointer;
    animation: map-marker-pulse 2.4s ease-in-out infinite;
  }

  @keyframes -global-map-marker-pulse {
    0%, 100% { box-shadow: 0 0 0 0   color-mix(in srgb, var(--color-accent, #00b37e) 40%, transparent); }
    50%       { box-shadow: 0 0 0 6px color-mix(in srgb, var(--color-accent, #00b37e)  0%, transparent); }
  }

  /* ── Popup styles (all MapLibre-injected DOM — always :global) ── */
  :global(.auction-popup .maplibregl-popup-content) {
    padding: 0;
    border-radius: 12px;
    border: 1px solid #e4e4e7;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
    overflow: hidden;
    background: #fff;
    font-family: var(--font-sans, system-ui, sans-serif);
  }
  :global(.auction-popup .maplibregl-popup-tip) { border-top-color: #fff; }

  :global(.popup-inner) {
    padding: 1rem 1.125rem 1.125rem;
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }
  :global(.popup-address) {
    font-size: 0.9375rem;
    font-weight: 700;
    color: #09090b;
    margin: 0;
    line-height: 1.3;
    letter-spacing: -0.02em;
  }
  :global(.popup-location) {
    font-size: 0.75rem;
    font-weight: 700;
    color: #71717a;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    font-family: var(--font-mono, monospace);
    margin: 0;
  }
  :global(.popup-meta) {
    display: flex;
    gap: 0.375rem;
    flex-wrap: wrap;
    margin-top: 0.25rem;
  }
  :global(.popup-chip) {
    font-family: var(--font-mono, monospace);
    font-size: 0.6875rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 999px;
    background: #f4f4f5;
    color: #3f3f46;
    white-space: nowrap;
  }
  :global(.popup-source) {
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: #a1a1aa;
    font-family: var(--font-mono, monospace);
    margin: 0;
  }
  :global(.popup-cta) {
    display: inline-block;
    margin-top: 0.5rem;
    font-size: 0.8125rem;
    font-weight: 700;
    color: var(--color-accent, #00b37e);
    text-decoration: none;
    letter-spacing: -0.01em;
  }
  :global(.popup-cta:hover) { text-decoration: underline; }

  :global([data-theme='dark'] .auction-popup .maplibregl-popup-content) {
    background: #18181b;
    border-color: #27272a;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
  }
  :global([data-theme='dark'] .auction-popup .maplibregl-popup-tip) { border-top-color: #18181b; }
  :global([data-theme='dark'] .popup-address) { color: #fafafa; }
  :global([data-theme='dark'] .popup-chip)    { background: #27272a; color: #a1a1aa; }
  :global([data-theme='dark'] .popup-source)  { color: #52525b; }
</style>
