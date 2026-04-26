import type { APIRoute } from 'astro';

const SITE = 'https://auctionandcompany.com';
const API  = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';

const staticPages = [
  { path: '/',       changefreq: 'daily',   priority: '1.0' },
  { path: '/about',  changefreq: 'monthly', priority: '0.5' },
];

export const GET: APIRoute = async () => {
  const today = new Date().toISOString().split('T')[0];

  // Static pages
  const staticUrls = staticPages.map(({ path, changefreq, priority }) => `
  <url>
    <loc>${SITE}${path}</loc>
    <lastmod>${today}</lastmod>
    <changefreq>${changefreq}</changefreq>
    <priority>${priority}</priority>
  </url>`);

  // Dynamic city landing pages — fetch all active slug pairs from the API
  let cityUrls: string[] = [];
  try {
    const res = await fetch(`${API}/auctions/slugs?limit=500`);
    if (res.ok) {
      const { markets } = await res.json() as { markets: { county_slug: string; city_slug: string }[] };
      cityUrls = markets.map(({ county_slug, city_slug }) => `
  <url>
    <loc>${SITE}/massachusetts/${county_slug}/${city_slug}</loc>
    <lastmod>${today}</lastmod>
    <changefreq>daily</changefreq>
    <priority>0.8</priority>
  </url>`);
    }
  } catch {
    // API unavailable at build time — city URLs omitted, not a fatal error.
  }

  const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${[...staticUrls, ...cityUrls].join('')}
</urlset>`;

  return new Response(body, {
    headers: {
      'Content-Type': 'application/xml',
      'Cache-Control': 'public, max-age=3600, s-maxage=3600',
    },
  });
};
