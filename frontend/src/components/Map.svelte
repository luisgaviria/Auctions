<script>
  import '../styles/map.css';
  import { onMount, onDestroy } from 'svelte';
  // `?url` tells Vite to copy the file to the build output and give us the hashed
  // URL as a string — without injecting the CSS into the page automatically.
  // We inject the <link> tag lazily in onMount so it never blocks the initial load.
  import maplibreCssUrl from 'maplibre-gl/dist/maplibre-gl.css?url';

  export let auctions = [];
  export let apiUrl = '';

  const MAPTILER_KEY = import.meta.env.VITE_MAPTILER_API_KEY;

  let mapContainer;
  let map = null;
  // Populated by the dynamic import inside onMount — null until first activation.
  let maplibregl = null;
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
    if (!map || !maplibregl) return;
    clearMarkers();

    auctionList.filter(validCoord).forEach(auction => {
      const lat = parseFloat(auction.lat);
      const lng = parseFloat(auction.lng);
      const lngLat = [lng, lat];

      const el = document.createElement('div');
      el.className = 'map-marker';

      const popupBtns = [
        auction.link          && `<a href="${auction.link}"           target="_blank" rel="noopener noreferrer" class="popup-btn popup-btn--listing">Listing</a>`,
        auction.zillow_url    && `<a href="${auction.zillow_url}"     target="_blank" rel="noopener noreferrer" class="popup-btn popup-btn--zillow">Zillow</a>`,
        auction.street_view_url && `<a href="${auction.street_view_url}" target="_blank" rel="noopener noreferrer" class="popup-btn popup-btn--maps">Maps</a>`,
        (auction.registry_deep_link || auction.registry_url) && `<a href="${auction.registry_deep_link || auction.registry_url}" target="_blank" rel="noopener noreferrer" class="popup-btn popup-btn--registry">Registry</a>`,
      ].filter(Boolean).join('');

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
          ${popupBtns ? `<div class="popup-actions">${popupBtns}</div>` : ''}
        </div>
      `);

      el.addEventListener('click', (e) => {
        e.stopPropagation();
        // Always fly to the tapped marker so it's centred on screen.
        map.flyTo({ center: lngLat, zoom: Math.max(map.getZoom(), 14), duration: 500 });

        // Always show the beautiful MapLibre popup regardless of screen size
        if (activePopup) activePopup.remove();
        popup.setLngLat(lngLat).addTo(map);
        activePopup = popup;

        // Notify the sidebar/sheet — include the full auction object so the
        // mobile sheet can render a complete card without a second fetch.
        mapContainer.dispatchEvent(new CustomEvent('markerclick', {
          bubbles: true,
          detail: { id: auction.id, auction },
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

  onMount(async () => {
    // Guard against HMR double-initialisation.
    if (map) return;

    // ── Lazy activation ────────────────────────────────────────────────────────
    // Don't load the 286 KiB MapLibre library until the user opens map view.
    // 'resizemap' is dispatched by setView() in index.astro after the container
    // is made visible (display:flex).  We block here so MapLibre is never
    // downloaded on a page load where the user stays in grid view.
    await new Promise(resolve => {
      mapContainer.addEventListener('resizemap', resolve, { once: true });
    });

    // ── Dynamic import (separate Vite chunk, only fetched on demand) ───────────
    maplibregl = (await import('maplibre-gl')).default;

    // ── Inject MapLibre CSS lazily ─────────────────────────────────────────────
    // maplibreCssUrl is a Vite ?url import: the file is in the build output but
    // the <link> tag is only added now, not on the initial page load.
    if (!document.getElementById('maplibre-gl-css')) {
      const link = document.createElement('link');
      link.id   = 'maplibre-gl-css';
      link.rel  = 'stylesheet';
      link.href = maplibreCssUrl;
      document.head.appendChild(link);
    }

    // ── Initialise map ─────────────────────────────────────────────────────────
    const maptilerKey = import.meta.env.VITE_MAPTILER_KEY;
    const mapStyle = maptilerKey 
      ? `https://api.maptiler.com/maps/dataviz-dark/style.json?key=${maptilerKey}`
      : {
          version: 8,
          sources: {
            'osm-tiles': {
              type: 'raster',
              tiles: ['https://a.tile.openstreetmap.org/{z}/{x}/{y}.png'],
              tileSize: 256,
              attribution: '© OpenStreetMap contributors'
            }
          },
          layers: [{
            id: 'osm-tiles-layer',
            type: 'raster',
            source: 'osm-tiles',
            minzoom: 0,
            maxzoom: 19
          }]
        };

    map = new maplibregl.Map({
      container: mapContainer,
      style: mapStyle,
      center: [-71.0589, 42.3601],
      zoom: 7,
      attributionControl: false
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

      // After the initial fit animation settles, auto-search the visible viewport
      // so the sidebar is populated immediately.  Then register the user-move tracker.
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
      if (!e.originalEvent.target.closest('.map-marker')) {
        if (activePopup) { activePopup.remove(); activePopup = null; }
        // On mobile, dismiss the carousel when tapping the map background.
        if (window.innerWidth < 768) {
          mapContainer.dispatchEvent(new CustomEvent('closemarkercarousel', { bubbles: true }));
        }
      }
    });

    // Subsequent resizemap events (user toggles back to map view after closing it).
    mapContainer.addEventListener('resizemap', () => map && map.resize());

    // 'flytomarker' is dispatched by the sidebar card click handler in index.astro.
    mapContainer.addEventListener('flytomarker', (e) => {
      const entry = markers.find(m => m.id === e.detail.id);
      if (!entry) return;

      if (activePopup) { activePopup.remove(); activePopup = null; }

      map.flyTo({ center: entry.lngLat, zoom: 15, duration: 700 });
      map.once('moveend', () => {
        entry.popup.setLngLat(entry.lngLat).addTo(map);
        activePopup = entry.popup;
      });
    });

    // 'pantomarker' is dispatched by the carousel scroll handler — pan the map
    // to centre a marker without opening its popup (no flyTo zoom change).
    mapContainer.addEventListener('pantomarker', (e) => {
      const entry = markers.find(m => m.id === e.detail.id);
      if (!entry) return;
      map.easeTo({ center: entry.lngLat, duration: 300 });
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
