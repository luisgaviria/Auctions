import { initState, state } from './state';
import { checkFavoriteStatus, initFavoritesClickHandler } from './favorites';
import { loadMore } from './load-more';
import { initDrawer } from './drawer';
import { populateMapCards, setView, showMarkerCarousel, hideMarkerCarousel } from './map-view';

// Hydrate shared state from the hidden #page-config element before anything else.
initState();

document.addEventListener('DOMContentLoaded', () => {
  checkFavoriteStatus();
  initFavoritesClickHandler();
  initDrawer();

  document.getElementById('load-more-btn')
    ?.addEventListener('click', (e) => loadMore(e.currentTarget as HTMLButtonElement));
  document.getElementById('map-load-more-btn')
    ?.addEventListener('click', (e) => loadMore(e.currentTarget as HTMLButtonElement));

  document.getElementById('view-toggle-btn')
    ?.addEventListener('click', () => setView(!state.isMapView));

  if (state.isMapView) populateMapCards(state.allAuctions);

  // Card click → fly map to marker.
  document.addEventListener('click', (e) => {
    const card = (e.target as Element).closest<HTMLElement>('.map-trigger-card');
    if (!card) return;
    e.preventDefault();
    e.stopPropagation();
    if ((e.target as Element).closest('.favorite-btn, .research-btn')) return;
    const id = parseInt(card.dataset.auctionId ?? '', 10);
    if (isNaN(id)) return;
    document.querySelector('.map-container')?.dispatchEvent(
      new CustomEvent('flytomarker', { bubbles: false, detail: { id } })
    );
  });

  // Marker click → mobile: show carousel; desktop: scroll & highlight sidebar card.
  document.getElementById('split-right')?.addEventListener('markerclick', (e: Event) => {
    const detail = (e as CustomEvent).detail;
    const id = detail?.id;
    if (!id) return;

    if (window.innerWidth < 768) {
      if (detail.auction) showMarkerCarousel(detail.auction);
      return;
    }

    const card = document.querySelector<HTMLElement>(
      `.map-trigger-card[data-auction-id="${id}"]`
    );
    if (!card) return;
    setTimeout(() => card.scrollIntoView({ behavior: 'smooth', block: 'center' }), 10);
    card.classList.remove('is-active-highlight');
    void card.offsetWidth;
    card.classList.add('is-active-highlight');
    setTimeout(() => card.classList.remove('is-active-highlight'), 2000);
  });

  // Close carousel via button or map background tap.
  document.getElementById('carousel-close')?.addEventListener('click', hideMarkerCarousel);
  document.getElementById('split-right')?.addEventListener('closemarkercarousel', hideMarkerCarousel);

  // Bbox search results → refresh sidebar cards.
  document.getElementById('split-right')?.addEventListener('auctionresultschange', (e: Event) => {
    const newAuctions: any[] = (e as CustomEvent).detail?.auctions ?? [];
    state.currentMapAuctions = newAuctions;
    populateMapCards(newAuctions);
    checkFavoriteStatus();
    const mapBtn = document.getElementById('map-load-more-btn');
    if (mapBtn) mapBtn.style.display = 'none';
  });
});
