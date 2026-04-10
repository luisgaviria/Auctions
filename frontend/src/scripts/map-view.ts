import { state } from './state';
import { createCardHTML } from './card';
import { checkFavoriteStatus } from './favorites';

function setContents(container: HTMLElement | null, html: string): void {
  if (!container) return;
  container.innerHTML = html;
  (container as HTMLElement & { style: CSSStyleDeclaration }).style.display  = 'flex';
  (container as HTMLElement & { style: CSSStyleDeclaration }).style.opacity  = '1';
  container.scrollTop = 0;
}

export function populateMapCards(auctionList: any[]): void {
  const mapList    = document.getElementById('map-card-list');
  const drawerList = document.getElementById('drawer-card-list');
  if (!mapList && !drawerList) return;

  if (!auctionList || auctionList.length === 0) {
    const empty = '<p class="sidebar-empty">No auctions found in this area.</p>';
    if (mapList)    mapList.innerHTML    = empty;
    if (drawerList) drawerList.innerHTML = empty;
    return;
  }

  const html = auctionList.map(createCardHTML).join('');
  setContents(mapList,    html);
  setContents(drawerList, html);
}

export function appendMapCards(newAuctions: any[]): void {
  const html = newAuctions.map(createCardHTML).join('');
  [document.getElementById('map-card-list'), document.getElementById('drawer-card-list')]
    .forEach(container => {
      if (!container) return;
      const temp = document.createElement('div');
      temp.innerHTML = html;
      while (temp.firstElementChild) container.appendChild(temp.firstElementChild);
    });
}

// ── Lazy map mount ─────────────────────────────────────────────────────────────
// Map.svelte + Svelte runtime (~19 KiB) are NOT included in the initial bundle.
// They are fetched only when the user opens map view for the first time.
let mapMounted = false;

async function ensureMapMounted(): Promise<void> {
  if (mapMounted) return;
  mapMounted = true;

  const target = document.getElementById('map-mount');
  if (!target) return;

  // Fetch both in parallel — Svelte runtime and the component chunk.
  const [{ mount }, { default: MapSvelte }] = await Promise.all([
    import('svelte'),
    import('../components/Map.svelte') as Promise<{ default: any }>,
  ]);

  mount(MapSvelte, {
    target,
    props: { auctions: state.allAuctions, apiUrl: state.apiUrl },
  });
}

// ── View toggle ────────────────────────────────────────────────────────────────
export async function setView(showMap: boolean): Promise<void> {
  state.isMapView = showMap;
  const gridView = document.getElementById('grid-view');
  const mapSplit = document.getElementById('map-split');
  const label    = document.getElementById('toggle-label');

  const auctionsPage = document.getElementById('auctions-page');

  if (showMap) {
    if (auctionsPage) auctionsPage.style.display = 'none';
    if (gridView) gridView.style.display = 'none';
    if (mapSplit) mapSplit.style.display = 'flex';
    document.getElementById('toggle-use')?.setAttribute('href', '#icon-th');
    if (label)    label.textContent = 'Grid View';

    populateMapCards(state.currentMapAuctions ?? state.allAuctions);
    checkFavoriteStatus();
    state.mapViewInitialised = true;

    const mapBtn    = document.getElementById('map-load-more-btn');
    const drawerBtn = document.getElementById('drawer-load-more-btn');
    [mapBtn, drawerBtn].forEach(b => {
      if (b) b.style.display = state.hasMore ? '' : 'none';
    });

    // Mount the Svelte component (no-op after first call), then signal the
    // map canvas to resize. The 50 ms delay gives Svelte's onMount time to
    // register the resizemap event listener before we fire it.
    await ensureMapMounted();
    setTimeout(() => {
      document.querySelector('.map-container')?.dispatchEvent(new CustomEvent('resizemap'));
    }, 50);
  } else {
    if (mapSplit) mapSplit.style.display = 'none';
    if (auctionsPage) auctionsPage.style.display = '';
    if (gridView) gridView.style.display = '';
    document.getElementById('toggle-use')?.setAttribute('href', '#icon-map');
    if (label)    label.textContent = 'Show Map';
  }
}
