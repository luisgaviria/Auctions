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
  const mapList = document.getElementById('map-card-list');
  if (!mapList) return;

  if (!auctionList || auctionList.length === 0) {
    mapList.innerHTML = '<p class="sidebar-empty">No auctions found in this area.</p>';
    return;
  }

  setContents(mapList, auctionList.map(createCardHTML).join(''));
}

let carouselScrollCleanup: (() => void) | null = null;

function buildDots(carousel: HTMLElement, track: HTMLElement, count: number): void {
  const dotsEl = document.getElementById('carousel-dots');
  if (!dotsEl) return;
  dotsEl.innerHTML = Array.from({ length: count }, (_, i) =>
    `<div class="carousel-dot${i === 0 ? ' active' : ''}"></div>`
  ).join('');

  const dots = dotsEl.querySelectorAll<HTMLElement>('.carousel-dot');
  const slides = track.querySelectorAll<HTMLElement>('.carousel-slide');

  const onScroll = () => {
    const slideWidth = track.offsetWidth + parseFloat(getComputedStyle(track).gap || '0') * 0 + 12;
    const idx = Math.round(track.scrollLeft / slideWidth);
    dots.forEach((d, i) => d.classList.toggle('active', i === idx));

    const slide = slides[idx];
    if (slide) {
      const id = parseInt(slide.dataset.auctionId ?? '', 10);
      if (!isNaN(id)) {
        document.querySelector('.map-container')?.dispatchEvent(
          new CustomEvent('pantomarker', { bubbles: false, detail: { id } })
        );
      }
    }
  };

  // Use IntersectionObserver per slide for accurate snap tracking
  const observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.intersectionRatio >= 0.6) {
          const slide = entry.target as HTMLElement;
          const id = parseInt(slide.dataset.auctionId ?? '', 10);
          const idx = Array.from(slides).indexOf(slide);
          dots.forEach((d, i) => d.classList.toggle('active', i === idx));
          if (!isNaN(id)) {
            document.querySelector('.map-container')?.dispatchEvent(
              new CustomEvent('pantomarker', { bubbles: false, detail: { id } })
            );
          }
        }
      }
    },
    { root: track, threshold: 0.6 }
  );

  slides.forEach(s => observer.observe(s));

  if (carouselScrollCleanup) carouselScrollCleanup();
  carouselScrollCleanup = () => {
    observer.disconnect();
    track.removeEventListener('scroll', onScroll);
  };
}

export function showMarkerCarousel(tappedAuction: any): void {
  const carousel = document.getElementById('marker-card-carousel');
  const track = document.getElementById('carousel-track');
  if (!carousel || !track) return;

  // Gather all current map auctions; put tapped one first
  const pool: any[] = state.currentMapAuctions ?? state.allAuctions ?? [];
  const others = pool.filter(a => a.id !== tappedAuction.id);
  const ordered = [tappedAuction, ...others.slice(0, 9)]; // max 10 slides

  track.innerHTML = ordered.map(a =>
    `<div class="carousel-slide" data-auction-id="${a.id}">${createCardHTML(a)}</div>`
  ).join('');

  // Re-scroll to first slide
  track.scrollLeft = 0;

  buildDots(carousel, track, ordered.length);

  carousel.dataset.state = 'visible';
}

export function hideMarkerCarousel(): void {
  const carousel = document.getElementById('marker-card-carousel');
  if (carousel) carousel.dataset.state = 'hidden';
  if (carouselScrollCleanup) { carouselScrollCleanup(); carouselScrollCleanup = null; }
}

export function appendMapCards(newAuctions: any[]): void {
  const container = document.getElementById('map-card-list');
  if (!container) return;
  const temp = document.createElement('div');
  temp.innerHTML = newAuctions.map(createCardHTML).join('');
  while (temp.firstElementChild) container.appendChild(temp.firstElementChild);
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

    const mapBtn = document.getElementById('map-load-more-btn');
    if (mapBtn) mapBtn.style.display = state.hasMore ? '' : 'none';

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
