import { state } from './state';
import { createCardHTML } from './card';
import { checkFavoriteStatus } from './favorites';
import { appendMapCards } from './map-view';
import { showToast } from './toast';

function todayET(): string {
  return new Date().toLocaleDateString('en-CA', { timeZone: 'America/New_York' });
}

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

  const topLoader = document.getElementById('top-loader');
  if (topLoader) {
    topLoader.classList.remove('done');
    topLoader.classList.add('loading');
  }

  try {
    const searchParam = state.search ? `&search=${encodeURIComponent(state.search)}` : '';
    const response = await fetch(`${state.apiUrl}/auctions?limit=${state.LIMIT}&offset=${state.offset}${searchParam}`);
    if (!response.ok) throw new Error('Server error');
    const data        = await response.json();
    const newAuctions: any[] = filterPastAuctions(data.auctions || []);

    const grid = document.getElementById('auctions-grid');
    const newElements: HTMLElement[] = [];

    newAuctions.forEach((a: any) => {
      const temp = document.createElement('div');
      temp.innerHTML = createCardHTML(a).trim();
      const el = temp.firstElementChild as HTMLElement | null;
      if (el && grid) {
        el.style.opacity = '0';
        el.style.transform = 'translateX(40px)';
        el.style.transition = 'none';
        grid.appendChild(el);
        newElements.push(el);
      }
    });

    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        newElements.forEach((el, i) => {
          el.style.transition = `opacity 0.5s cubic-bezier(0.16,1,0.3,1) ${i * 70}ms, transform 0.5s cubic-bezier(0.16,1,0.3,1) ${i * 70}ms`;
          el.style.opacity = '1';
          el.style.transform = 'translateX(0)';
          el.classList.add('revealed');
        });
      });
    });

    document.dispatchEvent(new CustomEvent('auctioncardsadded'));

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
    if (topLoader) {
      topLoader.classList.add('done');
      setTimeout(() => topLoader.classList.remove('loading', 'done'), 400);
    }
  }
}
