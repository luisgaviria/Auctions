import { experimental_AstroContainer as AstroContainer } from 'astro/container';
import { beforeAll, describe, expect, it } from 'vitest';
import Breadcrumb from '../../src/components/Breadcrumb.astro';
import Card from '../../src/components/Card.astro';
import ViewToggle from '../../src/components/ViewToggle.astro';
import Layout from '../../src/layouts/Layout.astro';

let container: Awaited<ReturnType<typeof AstroContainer.create>>;

beforeAll(async () => {
  container = await AstroContainer.create();
});

describe('Breadcrumb.astro', () => {
  it('renders linked ancestors, a current page, and matching JSON-LD', async () => {
    const crumbs = [
      { label: 'Massachusetts', href: '/massachusetts' },
      { label: 'Suffolk County', href: '/massachusetts/suffolk' },
      { label: 'Boston' },
    ];
    const html = await container.renderToString(Breadcrumb, { props: { crumbs } });

    expect(html).toContain('<nav class="breadcrumb" aria-label="Breadcrumb">');
    expect(html).toContain('<a href="/massachusetts">Massachusetts</a>');
    expect(html).toContain('<a href="/massachusetts/suffolk">Suffolk County</a>');
    expect(html).toContain('<span aria-current="page">Boston</span>');
    expect(html.match(/class="sep"/g)).toHaveLength(2);

    const schemaTag = html.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/);
    expect(schemaTag).not.toBeNull();
    expect(JSON.parse(schemaTag![1])).toMatchObject({
      '@type': 'BreadcrumbList',
      itemListElement: [
        { position: 1, name: 'Massachusetts', item: 'https://auctionandcompany.com/massachusetts' },
        {
          position: 2,
          name: 'Suffolk County',
          item: 'https://auctionandcompany.com/massachusetts/suffolk',
        },
        { position: 3, name: 'Boston' },
      ],
    });
  });
});

describe('Card.astro', () => {
  it('renders auction facts and structured listing data', async () => {
    const html = await container.renderToString(Card, {
      props: {
        id: 42,
        link: 'https://auctions.example.test/42',
        date: '2027-06-18',
        time: '10:00 AM',
        address: '12 Main Street',
        city: 'Boston',
        state: 'MA',
        status: 'Postponed',
        deposit: '$4,000',
        siteName: 'bayState',
      },
    });

    expect(html).toContain('data-status="postponed"');
    expect(html).toContain('12 Main Street');
    expect(html).toContain('Boston, MA');
    expect(html).toContain('Bay State');
    expect(html).toContain('title="View auction listing"');

    const schemaTag = html.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/);
    expect(schemaTag).not.toBeNull();
    expect(JSON.parse(schemaTag![1])).toMatchObject({
      '@type': 'RealEstateListing',
      name: '12 Main Street',
      price: 4000,
      priceCurrency: 'USD',
      availabilityStarts: '2027-06-18',
    });
  });

  it('renders accessible fallbacks for incomplete auction data', async () => {
    const html = await container.renderToString(Card, {
      props: {
        id: 7,
        link: '',
        date: '',
        time: '',
        address: '',
        city: '',
        state: '',
        status: '',
        deposit: '',
        siteName: '',
      },
    });

    expect(html).toContain('Address not available');
    expect(html).toContain('On Schedule');
    expect(html).toContain('aria-label="Add to favorites"');
    expect(html).toContain('Contact the auctioneer for deposit details.');
  });
});

describe('ViewToggle.astro', () => {
  it('renders the map toggle with an accessible label and default action text', async () => {
    const html = await container.renderToString(ViewToggle);

    expect(html).toContain('id="view-toggle-btn"');
    expect(html).toContain('aria-label="Toggle map view"');
    expect(html).toContain('id="toggle-label">Show Map</span>');
    expect(html).toContain('href="#icon-map"');
  });
});

describe('Layout.astro', () => {
  it('renders page metadata, noindex directives, and slotted content', async () => {
    const html = await container.renderToString(Layout, {
      props: {
        title: 'Boston',
        description: 'Upcoming Boston auctions',
        canonical: 'https://auctionandcompany.com/boston',
        ogImage: 'https://auctionandcompany.com/boston-og.png',
        noindex: true,
      },
      slots: { default: '<h1>Boston auctions</h1>' },
    });

    expect(html).toContain('<title>Boston | Auction &amp; Company</title>');
    expect(html).toContain('<meta name="description" content="Upcoming Boston auctions">');
    expect(html).toContain('<meta name="robots" content="noindex, nofollow">');
    expect(html).toContain('<link rel="canonical" href="https://auctionandcompany.com/boston">');
    expect(html).toContain(
      '<meta property="og:image" content="https://auctionandcompany.com/boston-og.png">'
    );
    expect(html).toContain('<h1>Boston auctions</h1>');
  });

  it('uses the home title, default description, and indexable robots policy', async () => {
    const html = await container.renderToString(Layout, {
      props: { title: 'Auctions' },
    });

    expect(html).toContain(
      '<title>Upcoming Property Auctions in Massachusetts | Auction &amp; Company</title>'
    );
    expect(html).toContain(
      '<meta name="description" content="Browse upcoming real estate auctions in Massachusetts. Live bids, dates, deposits, and auctioneer details — updated daily.">'
    );
    expect(html).toContain('<meta name="robots" content="index, follow">');
    expect(html).toContain('<meta property="og:image" content="https://example.com/og.png">');
  });
});
