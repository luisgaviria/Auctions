import { state } from './state';
import { createCardHTML } from './card';
import { checkFavoriteStatus } from './favorites';
import { appendMapCards } from './map-view';
import { showToast } from './toast';

// todayET returns today's date in America/New_York as a "YYYY-MM-DD" string.
function todayET(): string {
  return new Date().toLocaleDateString('en-CA', { timeZone: 'America/New_York' });
}

// filterPastAuctions is a client-side safety net that removes auctions whose
// date has already passed in Eastern Time.  The API should already exclude
// them, but this guard prevents stale cached responses from reaching the UI.
function filterPastAuctions(list: any[]): any[] {
  const today = todayET();
  return list.filter((a) => {
    if (!a.date) return true;
    const parsed = new Date(a.date);
    if (isNaN(parsed.getTime())) return true;
    return parsed.toLocaleDateString('en-CA') >= today;
  });
}

export async function loadMore(triggerBtn: HTMLButtonElement): Promise<void> {
  if (state.loading || !state.hasMore) return;
  state.loading = true;

  const origText = triggerBtn.textContent ?? 'Load More';
  triggerBtn.textContent = 'Loading…';
  triggerBtn.disabled    = true;

  try {
    const response = await fetch(`${state.apiUrl}/auctions?limit=${state.LIMIT}&offset=${state.offset}`);
    if (!response.ok) throw new Error('Server error');
    const data        = await response.json();
    const newAuctions: any[] = filterPastAuctions(data.auctions || []);

    const grid = document.getElementById('auctions-grid');
    newAuctions.forEach((a: any) => {
      const temp = document.createElement('div');
      temp.innerHTML = createCardHTML(a).trim();
      if (temp.firstElementChild) grid?.appendChild(temp.firstElementChild);
    });

    if (state.mapViewInitialised) appendMapCards(newAuctions);

    state.allAuctions = [...state.allAuctions, ...newAuctions];
    state.offset     += newAuctions.length;

    if (newAuctions.length < state.LIMIT) {
      state.hasMore = false;
      ['load-more-btn', 'map-load-more-btn', 'drawer-load-more-btn'].forEach(id => {
        const b = document.getElementById(id);
        if (b) b.style.display = 'none';
      });
    }

    await checkFavoriteStatus();
  } catch {
    showToast('Failed to load more auctions', 'error');
  } finally {
    state.loading = false;
    if (state.hasMore) {
      triggerBtn.textContent = origText;
      triggerBtn.disabled    = false;
    }
  }
}
