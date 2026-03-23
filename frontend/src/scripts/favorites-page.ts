import { getStatusCategory, formatSiteName } from './card';

/** Reads apiUrl from the hidden #favorites-config element. */
function getApiUrl(): string {
  const el = document.getElementById('favorites-config') as HTMLElement | null;
  return el?.dataset.apiUrl ?? '';
}

function createFavoriteCardHTML(auction: any): string {
  const date     = auction.date    || '—';
  const time     = auction.time    || '';
  const deposit  = auction.deposit || '—';
  const address  = auction.address || 'Address not available';
  const status   = auction.status  || 'On Schedule';
  const safeLink    = (auction.link || '#').replace(/"/g, '&quot;');
  const safeAddress = address.replace(/"/g, '&quot;');
  const statusCat   = getStatusCategory(status);
  const siteName    = formatSiteName(auction.site_name || '');

  const locationLine = auction.city && auction.state
    ? `<p class="location-line">${auction.city}, ${auction.state}</p>` : '';
  const auctioneerHtml = siteName
    ? `<span class="auctioneer-label">${siteName}</span>` : '';

  return `
    <a href="${safeLink}" target="_blank" rel="noopener noreferrer"
       class="auction-card" aria-label="View auction: ${safeAddress}">
      <div class="card-spotlight" aria-hidden="true"></div>
      <div class="card-inner">
        <div class="card-header">
          <div class="status-indicator" data-status="${statusCat}">
            <span class="status-dot"></span>
            <span class="status-text">${status}</span>
          </div>
          ${auctioneerHtml}
        </div>
        <div class="card-address-block">
          <h3 class="property-address">${address}</h3>
          ${locationLine}
        </div>
        <div class="card-data-row">
          <div class="data-cell">
            <span class="data-value">${date}</span>
            <span class="data-label">Date</span>
          </div>
          <div class="data-cell">
            <span class="data-value${time ? '' : ' data-value--tbd'}">${time || 'TBD'}</span>
            <span class="data-label">Time</span>
          </div>
          <div class="data-cell">
            <span class="data-value">${deposit}</span>
            <span class="data-label">Deposit</span>
          </div>
        </div>
        <div class="card-footer">
          <button class="favorite-btn"
            data-auction-id="${auction.id}"
            data-loading="false"
            title="Remove from favorites"
            aria-pressed="true"
            aria-label="Remove from favorites">
            <i class="fas fa-heart"></i>
            <div class="fav-spinner"></div>
          </button>
        </div>
      </div>
    </a>`;
}

async function loadFavorites(): Promise<void> {
  if (!localStorage.getItem('jwt_token')) {
    window.location.href = '/login';
    return;
  }

  const apiUrl = getApiUrl();
  const container = document.getElementById('favorites-container');
  if (!container) return;

  container.innerHTML = `
    <div class="loading-state">
      <div class="page-spinner"></div>
      <p>Loading favorites…</p>
    </div>`;

  try {
    const response = await fetch(`${apiUrl}/favorites`, {
      headers: { Authorization: `Bearer ${localStorage.getItem('jwt_token')}` },
      credentials: 'include',
    });

    if (!response.ok) throw new Error('Failed to load favorites');

    const data = await response.json();

    if (data.auctions.length === 0) {
      container.innerHTML = `
        <div class="empty-state">
          <i class="far fa-heart"></i>
          <h2>No favorites yet</h2>
          <p>Mark auctions as favorites to track them here.</p>
          <a href="/" class="btn btn-primary">Browse Auctions</a>
        </div>`;
      return;
    }

    container.innerHTML = data.auctions.map(createFavoriteCardHTML).join('');
  } catch {
    container.innerHTML = `
      <div class="error-state">
        <i class="fas fa-exclamation-circle"></i>
        <h2>Error loading favorites</h2>
        <p>Something went wrong. Please try again.</p>
        <button onclick="window.location.reload()" class="btn btn-primary">Try Again</button>
      </div>`;
  }
}

/** Favorite removal via event delegation. */
function initFavoriteRemoval(): void {
  const apiUrl = getApiUrl();

  document.addEventListener('click', async (e) => {
    const button = (e.target as Element).closest('.favorite-btn') as HTMLElement | null;
    if (!button) return;

    e.preventDefault();
    e.stopPropagation();

    if (!localStorage.getItem('jwt_token')) {
      window.location.href = '/login';
      return;
    }

    if (button.dataset.loading === 'true') return;

    const auctionId = parseInt(button.dataset.auctionId ?? '', 10);
    if (isNaN(auctionId)) return;

    const icon = button.querySelector('i');
    const isFavorited = icon?.classList.contains('fas') ?? false;

    try {
      button.dataset.loading = 'true';
      const endpoint = isFavorited ? 'remove' : 'add';
      const response = await fetch(`${apiUrl}/favorites/${endpoint}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('jwt_token')}`,
        },
        body: JSON.stringify({ auction_id: auctionId }),
        credentials: 'include',
      });

      if (response.ok && isFavorited) {
        const card = button.closest('.auction-card');
        if (card) card.remove();
        const remaining = document.querySelectorAll('.auction-card');
        if (remaining.length === 0) loadFavorites();
      }
    } catch {
      /* silently ignore */
    } finally {
      button.dataset.loading = 'false';
    }
  });
}

document.addEventListener('DOMContentLoaded', () => {
  initFavoriteRemoval();
  loadFavorites();
});
