import { afterEach, describe, expect, it, vi } from 'vitest';
import { initDrawer } from '../../src/scripts/drawer';

type TestEventListener = (event: Event) => void;

function createElementMock(initialState = '') {
  const listeners = new Map<string, TestEventListener>();

  return {
    dataset: { state: initialState },
    textContent: '',
    addEventListener: vi.fn((type: string, listener: TestEventListener) => {
      listeners.set(type, listener);
    }),
    dispatch(type: string, event: Event): void {
      listeners.get(type)?.(event);
    },
  };
}

function touchEvent(type: string, property: 'touches' | 'changedTouches', clientY: number): Event {
  const event = new Event(type);
  Object.defineProperty(event, property, { value: [{ clientY }] });
  return event;
}

function installDrawerElements() {
  const drawer = createElementMock('peek');
  const toggleButton = createElementMock();
  const label = createElementMock();
  const elements: Record<string, ReturnType<typeof createElementMock>> = {
    'mobile-drawer': drawer,
    'drawer-toggle': toggleButton,
    'drawer-label': label,
  };

  vi.stubGlobal('document', {
    getElementById: (id: string) => elements[id] ?? null,
  });

  return { drawer, toggleButton, label };
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('initDrawer', () => {
  it('toggles between peek and expanded states with matching labels', () => {
    const { drawer, toggleButton, label } = installDrawerElements();
    initDrawer();

    toggleButton.dispatch('click', new Event('click'));
    expect(drawer.dataset.state).toBe('expanded');
    expect(label.textContent).toBe('Hide listings');

    toggleButton.dispatch('click', new Event('click'));
    expect(drawer.dataset.state).toBe('peek');
    expect(label.textContent).toBe('Show listings');
  });

  it('expands and collapses after vertical swipes beyond the threshold', () => {
    const { drawer, label } = installDrawerElements();
    initDrawer();

    drawer.dispatch('touchstart', touchEvent('touchstart', 'touches', 120));
    drawer.dispatch('touchend', touchEvent('touchend', 'changedTouches', 70));
    expect(drawer.dataset.state).toBe('expanded');
    expect(label.textContent).toBe('Hide listings');

    drawer.dispatch('touchstart', touchEvent('touchstart', 'touches', 70));
    drawer.dispatch('touchend', touchEvent('touchend', 'changedTouches', 120));
    expect(drawer.dataset.state).toBe('peek');
    expect(label.textContent).toBe('Show listings');
  });

  it('does nothing when required drawer elements are missing', () => {
    vi.stubGlobal('document', { getElementById: () => null });

    expect(() => initDrawer()).not.toThrow();
  });
});
