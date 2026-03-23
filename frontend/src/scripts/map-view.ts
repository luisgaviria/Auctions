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

export function setView(showMap: boolean): void {
  state.isMapView = showMap;
  const gridView = document.getElementById('grid-view');
  const mapSplit = document.getElementById('map-split');
  const icon     = document.getElementById('toggle-icon');
  const label    = document.getElementById('toggle-label');

  if (showMap) {
    if (gridView) gridView.style.display = 'none';
    if (mapSplit) mapSplit.style.display = 'flex';
    if (icon)     icon.className  = 'fas fa-th';
    if (label)    label.textContent = 'Grid View';

    setTimeout(() => {
      document.querySelector('.map-container')?.dispatchEvent(new CustomEvent('resizemap'));
    }, 50);

    populateMapCards(state.currentMapAuctions ?? state.allAuctions);
    checkFavoriteStatus();
    state.mapViewInitialised = true;

    const mapBtn    = document.getElementById('map-load-more-btn');
    const drawerBtn = document.getElementById('drawer-load-more-btn');
    [mapBtn, drawerBtn].forEach(b => {
      if (b) b.style.display = state.hasMore ? '' : 'none';
    });
  } else {
    if (mapSplit) mapSplit.style.display = 'none';
    if (gridView) gridView.style.display = '';
    if (icon)     icon.className  = 'fas fa-map';
    if (label)    label.textContent = 'Show Map';
  }
}
