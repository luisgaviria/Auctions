export function getStatusCategory(s: string): 'active' | 'postponed' | 'muted' {
  const lower = (s || '').toLowerCase();
  if (lower.includes('postponed')) return 'postponed';
  if (!s || lower === 'date tbd' || lower === 'tbd') return 'muted';
  return 'active';
}

export function formatSiteName(s: string): string {
  if (!s) return '';
  return s
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .split(/[\s_-]+/)
    .map((w: string) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join(' ');
}

export function createCardHTML(auction: any): string {
  const date = auction.date || '—';
  const time = auction.time || '';
  const deposit = auction.deposit || '—';
  const address = auction.address || 'Address not available';
  const status = auction.status || 'On Schedule';
  const safeLink = (auction.link || '#').replace(/"/g, '&quot;');
  const safeAddress = address.replace(/"/g, '&quot;');
  const statusCat = getStatusCategory(status);
  const siteName = formatSiteName(auction.site_name || '');
  const locationLine =
    auction.city && auction.state
      ? `<p class="location-line">${auction.city}, ${auction.state}</p>`
      : '';
  const auctioneerHtml = siteName ? `<span class="auctioneer-label">${siteName}</span>` : '';

  return `
    <a href="#"
       class="auction-card map-trigger-card"
       data-auction-id="${auction.id}"
       data-link="${safeLink}"
       aria-label="Fly to auction: ${safeAddress}">
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
          <div class="property-address">${address}</div>
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
            title="Add to favorites"
            aria-pressed="false"
            aria-label="Add to favorites">
            <svg class="icon" aria-hidden="true"><use href="#icon-heart-o"></use></svg>
            <div class="fav-spinner"></div>
          </button>
        </div>
      </div>
    </a>`;
}
