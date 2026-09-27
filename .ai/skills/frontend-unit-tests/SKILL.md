---
name: frontend-unit-tests
description: 'Use when writing, debugging, reviewing, or running unit tests for this repository’s Astro frontend with Vitest, including TypeScript browser scripts, Astro components, state, API-facing behavior, accessibility markup, and client interactions.'
---

# Frontend Unit Tests

Use this skill for isolated tests under `frontend/tests/` and for testable frontend behavior in `frontend/src/`. Keep tests deterministic and compatible with the repository’s Vitest configuration and custom Astro transform.

## Test Selection

1. Find the nearest existing test for the behavior: `tests/scripts/` for browser utilities and state, or `tests/components/` for Astro rendering.
2. Run one test file or test name first, then the frontend suite. Use the production build when changes affect Astro compilation, TypeScript, routes, or integration contracts.

From `frontend/`:

```sh
npm test -- tests/scripts/state.test.ts
npm test -- tests/scripts/state.test.ts -t 'hydrates page settings'
npm test
npm run build
```

## Test Patterns

- Use Vitest imports (`describe`, `it`, `expect`, `beforeAll`, `afterEach`, and `vi`) and keep test names focused on observable behavior.
- Test pure TypeScript helpers with representative normal, empty, null/optional, malformed, and boundary inputs. Prefer exact structured assertions over snapshots unless markup structure is the behavior being protected.
- For shared browser state, reset the exported state after each test and use `vi.stubGlobal`/`vi.unstubAllGlobals` for `document`, `window`, or other browser APIs. Do not leak listeners, timers, mocks, or mutated globals across tests.
- For Astro components, use `experimental_AstroContainer` from `astro/container`, create the container once with `beforeAll`, render with explicit props, and assert semantic markup, accessible fallbacks, links, attributes, and JSON-LD where applicable.
- Mock network requests, timers, MapLibre, and browser-only APIs at the boundary. Do not call the real backend, map provider, or external websites from unit tests.
- Match the existing API contract when testing serialized auction data: account for absent/null fields and the backend’s string-formatted dates, deposits, and coordinates.
- Cover failure and incomplete-data paths, including missing `#page-config`, invalid JSON, optional links, empty results, and navigation or event behavior where the module owns it.

## Test Hygiene

Keep fixtures small and local to the test file unless a shared helper already exists. Avoid depending on test order, current date/time, DOM layout, or production environment variables. When a test must inspect generated HTML, assert stable semantic markers rather than incidental whitespace or compiler output.

## Completion Check

Run the focused Vitest command after the first change. Then run `npm test` and `npm run build` for broader validation when the change affects shared frontend behavior, Astro components, or TypeScript compilation. Use `npm run dev` and browser verification only for behavior that cannot be meaningfully covered by unit tests.
