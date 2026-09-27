import { describe, expect, it } from 'vitest';
import { createCardHTML, formatSiteName, getStatusCategory } from '../../src/scripts/card';

describe('getStatusCategory', () => {
  it.each([
    ['Postponed by lender', 'postponed'],
    ['Date TBD', 'muted'],
    ['TBD', 'muted'],
    ['On Schedule', 'active'],
  ] as const)('categorizes status %j as %s', (status, category) => {
    expect(getStatusCategory(status)).toBe(category);
  });
});

describe('formatSiteName', () => {
  it('normalizes camel case and word separators', () => {
    expect(formatSiteName('bayState-auction_site')).toBe('Bay State Auction Site');
  });

  it('returns an empty string when no site name is provided', () => {
    expect(formatSiteName('')).toBe('');
  });
});

describe('createCardHTML', () => {
  it('renders auction details, status, location, and available research links', () => {
    const html = createCardHTML({
      id: 42,
      link: 'https://auctions.example.test/42',
      date: 'Jun 18, 2027',
      time: '10:00 AM',
      deposit: '$4,000',
      address: '12 Main Street',
      city: 'Boston',
      state: 'MA',
      status: 'Postponed',
      site_name: 'bayState',
      zillow_url: 'https://zillow.example.test/42',
    });

    expect(html).toContain('data-auction-id="42"');
    expect(html).toContain('data-status="postponed"');
    expect(html).toContain('12 Main Street');
    expect(html).toContain('Boston, MA');
    expect(html).toContain('Bay State');
    expect(html).toContain('Jun 18, 2027');
    expect(html).toContain('10:00 AM');
    expect(html).toContain('$4,000');
    expect(html).toContain('title="View auction listing"');
    expect(html).toContain('title="View on Zillow"');
    expect(html).not.toContain('research-btn--streetview');
  });

  it('uses fallbacks and omits research links when optional data is absent', () => {
    const html = createCardHTML({ id: 7 });

    expect(html).toContain('Address not available');
    expect(html).toContain('On Schedule');
    expect(html).toContain('data-value--tbd">TBD');
    expect(html).toContain('aria-label="Auction: Address not available"');
    expect(html).not.toContain('research-btn--auction');
    expect(html).not.toContain('research-btn--zillow');
    expect(html).not.toContain('research-btn--streetview');
    expect(html).not.toContain('research-btn--registry');
  });

  it('prefers a registry deep link when both registry URLs are present', () => {
    const deepLink = 'https://registry.example.test/deep';
    const fallbackLink = 'https://registry.example.test/property';
    const html = createCardHTML({
      id: 9,
      registry_deep_link: deepLink,
      registry_url: fallbackLink,
    });

    expect(html).toContain(`window.open('${deepLink}'`);
    expect(html).not.toContain(`window.open('${fallbackLink}'`);
  });
});
