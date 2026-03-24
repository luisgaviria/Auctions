import { state } from './state';
import { showToast } from './toast';

export async function checkFavoriteStatus(): Promise<void> {
  if (!localStorage.getItem('jwt_token')) return;
  try {
    const response = await fetch(`${state.apiUrl}/favorites`, {
      headers: { Authorization: `Bearer ${localStorage.getItem('jwt_token')}` },
      credentials: 'include',
    });
    if (!response.ok) return;
    const data = await response.json();
    const favoriteIds = new Set<number>((data.auctions || []).map((a: any) => a.id));
    document.querySelectorAll<HTMLElement>('.favorite-btn').forEach(btn => {
      const id = parseInt(btn.dataset.auctionId ?? '', 10);
      const useEl = btn.querySelector('use');
      if (!useEl) return;
      if (favoriteIds.has(id)) {
        useEl.setAttribute('href', '#icon-heart');
      } else {
        useEl.setAttribute('href', '#icon-heart-o');
      }
    });
  } catch { /* silently ignore */ }
}

export function initFavoritesClickHandler(): void {
  document.addEventListener('click', async (e) => {
    const button = (e.target as Element).closest<HTMLButtonElement>('.favorite-btn');
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

    const useEl = button.querySelector('use');
    if (!useEl) return;
    const isFavorited = useEl.getAttribute('href') === '#icon-heart';

    try {
      button.dataset.loading = 'true';
      const endpoint = isFavorited ? 'remove' : 'add';
      const response = await fetch(`${state.apiUrl}/favorites/${endpoint}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('jwt_token')}`,
        },
        body: JSON.stringify({ auction_id: auctionId }),
        credentials: 'include',
      });
      if (response.ok) {
        document.querySelectorAll(`.favorite-btn[data-auction-id="${auctionId}"]`).forEach(btn => {
          const u = btn.querySelector('use');
          if (!u) return;
          u.setAttribute('href', isFavorited ? '#icon-heart-o' : '#icon-heart');
          btn.setAttribute('aria-pressed', String(!isFavorited));
        });
        showToast(isFavorited ? 'Removed from favorites' : 'Added to favorites', isFavorited ? 'info' : 'success');
      } else {
        showToast('Error updating favorites', 'error');
      }
    } catch {
      showToast('Network error occurred', 'error');
    } finally {
      button.dataset.loading = 'false';
    }
  });
}
