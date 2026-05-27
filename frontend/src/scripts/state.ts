export interface PageState {
  apiUrl: string;
  LIMIT: number;
  allAuctions: any[];
  offset: number;
  hasMore: boolean;
  loading: boolean;
  isMapView: boolean;
  mapViewInitialised: boolean;
  currentMapAuctions: any[] | null;
  search: string;
}

export const state: PageState = {
  apiUrl: '',
  LIMIT: 20,
  allAuctions: [],
  offset: 0,
  hasMore: false,
  loading: false,
  isMapView: false,
  mapViewInitialised: false,
  currentMapAuctions: null,
  search: '',
};

/** Hydrates state from the hidden #page-config element written by index.astro. */
export function initState(): void {
  const el = document.getElementById('page-config') as HTMLElement | null;
  if (!el) return;
  state.apiUrl  = el.dataset.apiUrl  ?? '';
  state.LIMIT   = parseInt(el.dataset.limit ?? '20', 10);
  state.hasMore = el.dataset.hasMore === 'true';
  try {
    state.allAuctions = JSON.parse(el.dataset.initialAuctions ?? '[]');
  } catch {
    state.allAuctions = [];
  }
  state.offset = state.allAuctions.length;
}
