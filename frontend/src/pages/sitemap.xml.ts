import type { APIRoute } from 'astro';

const SITE = 'https://auctionandcompany.com';

const pages = [
  { path: '/',       changefreq: 'daily',   priority: '1.0' },
  { path: '/about',  changefreq: 'monthly', priority: '0.5' },
];

export const GET: APIRoute = () => {
  const urls = pages
    .map(
      ({ path, changefreq, priority }) => `
  <url>
    <loc>${SITE}${path}</loc>
    <changefreq>${changefreq}</changefreq>
    <priority>${priority}</priority>
  </url>`
    )
    .join('');

  const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls}
</urlset>`;

  return new Response(body, {
    headers: {
      'Content-Type': 'application/xml',
      'Cache-Control': 'public, max-age=3600, s-maxage=3600',
    },
  });
};
