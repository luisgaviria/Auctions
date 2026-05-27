import { state } from './state';
import { createCardHTML } from './card';
import { checkFavoriteStatus } from './favorites';
import { appendMapCards, clearMapCards } from './map-view';
import { showToast } from './toast';

let debounceTimer: ReturnType<typeof setTimeout> | null = null;

function todayET(): string {
  return new Date().toLocaleDateString('en-CA', { timeZone: 'America/New_York' });
}

function filterPast(list: any[]): any[] {
  const today = todayET();
  return list.filter(a => {
    if (!a.date) return true;
    const parsed = new Date(a.date);
    if (isNaN(parsed.getTime())) return true;
    return parsed.toLocaleDateString('en-CA') >= today;
  });
}

async function runSearch(query: string): Promise<void> {
  if (state.loading) return;
  state.loading = true;
  state.search  = query.trim();

  const grid = document.getElementById('auctions-grid');
  const loadMoreBtns = ['load-more-btn', 'map-load-more-btn', 'drawer-load-more-btn']
    .map(id => document.getElementById(id) as HTMLButtonElement | null);

  // Show skeleton while fetching.
  if (grid) grid.style.opacity = '0.4';

  try {
    const url = state.search
      ? `${state.apiUrl}/auctions?limit=${state.LIMIT}&offset=0&search=${encodeURIComponent(state.search)}`
      : `${state.apiUrl}/auctions?limit=${state.LIMIT}&offset=0`;

    const res = await fetch(url);
    if (!res.ok) throw new Error('Server error');
    const data = await res.json();
    const auctions: any[] = filterPast(data.auctions || []);

    // Replace grid content.
    if (grid) {
      grid.innerHTML = '';
      if (auctions.length === 0) {
        grid.innerHTML = `<div class="empty-state" style="grid-column:1/-1">
          <p class="text-secondary">No auctions found${state.search ? ` for "${state.search}"` : ''}.</p>
        </div>`;
      } else {
        auctions.forEach(a => {
          const tmp = document.createElement('div');
          tmp.innerHTML = createCardHTML(a).trim();
          if (tmp.firstElementChild) grid.appendChild(tmp.firstElementChild);
        });
      }
    }

    // Reset pagination state.
    state.allAuctions = auctions;
    state.offset      = auctions.length;
    state.hasMore     = auctions.length === state.LIMIT;

    loadMoreBtns.forEach(btn => {
      if (btn) btn.style.display = state.hasMore ? '' : 'none';
    });

    if (state.mapViewInitialised) {
      clearMapCards();
      appendMapCards(auctions);
    }

    await checkFavoriteStatus();
  } catch {
    showToast('Search failed — please try again', 'error');
  } finally {
    state.loading = false;
    if (grid) grid.style.opacity = '';
  }
}

export function initSearch(): void {
  const input = document.getElementById('auction-search') as HTMLInputElement | null;
  if (!input) return;

  input.addEventListener('input', () => {
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => runSearch(input.value), 380);
  });

  // Clear button (×).
  const clearBtn = document.getElementById('search-clear-btn');
  clearBtn?.addEventListener('click', () => {
    input.value = '';
    input.focus();
    runSearch('');
  });

  // Show/hide clear button as user types.
  input.addEventListener('input', () => {
    if (clearBtn) clearBtn.style.display = input.value ? '' : 'none';
  });
}
