import { afterEach, describe, expect, it, vi } from 'vitest';
import { initState, state } from '../../src/scripts/state';

function setPageConfig(dataset: Record<string, string> | null): void {
  vi.stubGlobal('document', {
    getElementById: vi.fn().mockReturnValue(dataset ? { dataset } : null),
  });
}

function resetState(): void {
  Object.assign(state, {
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
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
  resetState();
});

describe('initState', () => {
  it('hydrates page settings and auctions from the config element', () => {
    const auctions = [{ id: 1 }, { id: 2 }];
    setPageConfig({
      apiUrl: 'https://api.example.test',
      limit: '12',
      hasMore: 'true',
      initialAuctions: JSON.stringify(auctions),
    });

    initState();

    expect(state).toMatchObject({
      apiUrl: 'https://api.example.test',
      LIMIT: 12,
      hasMore: true,
      allAuctions: auctions,
      offset: 2,
    });
  });

  it('uses defaults when optional config is missing or auction JSON is invalid', () => {
    setPageConfig({ initialAuctions: '{invalid' });

    initState();

    expect(state).toMatchObject({
      apiUrl: '',
      LIMIT: 20,
      hasMore: false,
      allAuctions: [],
      offset: 0,
    });
  });

  it('leaves state unchanged when the config element is absent', () => {
    state.apiUrl = 'https://api.example.test';
    state.offset = 7;
    setPageConfig(null);

    initState();

    expect(state.apiUrl).toBe('https://api.example.test');
    expect(state.offset).toBe(7);
  });
});
