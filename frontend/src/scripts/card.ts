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

  // Research buttons must be <button>, NOT <a>, because nesting <a> inside
  // the outer auction-card <a> is invalid HTML — the browser auto-closes the
  // outer anchor, breaking the entire card structure.
  const safeOpen = (url: string) =>
    `event.stopPropagation();window.open('${url.replace(/'/g, "\\'")}','_blank','noopener,noreferrer')`;

  const zillowBtn = auction.zillow_url
    ? `<button type="button" class="research-btn research-btn--zillow" title="View on Zillow" aria-label="View on Zillow" onclick="${safeOpen(auction.zillow_url)}">
        <svg aria-hidden="true" viewBox="0 0 24 24" fill="currentColor" width="14" height="14"><path d="M12 2L2 9.5l1.5 1L12 4.3l8.5 6.2 1.5-1L12 2zm0 3.8L4 11.5V21h6v-6h4v6h6V11.5L12 5.8z"/></svg>
        <span>Zillow</span>
      </button>`
    : '';
  const streetViewBtn = auction.street_view_url
    ? `<button type="button" class="research-btn research-btn--streetview" title="View on Google Maps" aria-label="View on Google Maps" onclick="${safeOpen(auction.street_view_url)}">
        <svg aria-hidden="true" viewBox="0 0 24 24" fill="currentColor" width="14" height="14"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/></svg>
        <span>Maps</span>
      </button>`
    : '';
  const registryTarget = auction.registry_deep_link || auction.registry_url;
  const registryBtn = registryTarget
    ? `<button type="button" class="research-btn research-btn--registry" title="View Registry of Deeds" aria-label="View Registry of Deeds" onclick="${safeOpen(registryTarget)}">
        <svg aria-hidden="true" viewBox="0 0 24 24" fill="currentColor" width="14" height="14"><path d="M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm4 18H6V4h7v5h5v11zM8 15h8v2H8zm0-4h8v2H8z"/></svg>
        <span>Registry</span>
      </button>`
    : '';
  const listingBtn = auction.link
    ? `<button type="button" class="research-btn research-btn--auction" title="View auction listing" aria-label="View auction listing" onclick="${safeOpen(auction.link)}">
        <svg aria-hidden="true" viewBox="0 0 24 24" fill="currentColor" width="14" height="14"><path d="M19 19H5V5h7V3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7h-2v7zM14 3v2h3.59l-9.83 9.83 1.41 1.41L19 6.41V10h2V3h-7z"/></svg>
        <span>Listing</span>
      </button>`
    : '';
  const toolbarHtml = `<div class="card-research">${listingBtn}${zillowBtn}${streetViewBtn}${registryBtn}</div>`;

  return `
    <div class="auction-card map-trigger-card"
         data-auction-id="${auction.id}"
         data-link="${safeLink}"
         aria-label="Auction: ${safeAddress}">
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
      ${toolbarHtml}
    </div>`;
}
