import type { APIRoute } from 'astro';
import satori from 'satori';
import { Resvg } from '@resvg/resvg-js';

// Fetch Inter font once per cold start.
let fontData: ArrayBuffer | null = null;
async function getFont(): Promise<ArrayBuffer> {
  if (fontData) return fontData;
  const res = await fetch(
    'https://fonts.gstatic.com/s/inter/v13/UcCO3FwrK3iLTeHuS_fvQtMwCp50KnMw2boKoduKmMEVuLyfAZ9hiJ-Ek-_EeA.woff'
  );
  fontData = await res.arrayBuffer();
  return fontData;
}

export const GET: APIRoute = async () => {
  const font = await getFont();

  const svg = await satori(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ({
      type: 'div',
      props: {
        style: {
          width: '1200px',
          height: '630px',
          background: '#09090b',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          padding: '64px 72px',
          fontFamily: 'Inter, sans-serif',
          position: 'relative',
        },
        children: [
          // Top row: badge + "Auction & Company"
          {
            type: 'div',
            props: {
              style: {
                display: 'flex',
                alignItems: 'center',
                gap: '20px',
              },
              children: [
                // AC badge
                {
                  type: 'div',
                  props: {
                    style: {
                      width: '56px',
                      height: '56px',
                      background: '#fafafa',
                      borderRadius: '10px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: '18px',
                      fontWeight: '800',
                      color: '#09090b',
                      letterSpacing: '0.04em',
                    },
                    children: 'AC',
                  },
                },
                // Wordmark
                {
                  type: 'div',
                  props: {
                    style: {
                      fontSize: '22px',
                      fontWeight: '700',
                      color: '#a1a1aa',
                      letterSpacing: '0.12em',
                      textTransform: 'uppercase',
                    },
                    children: 'Auction & Company',
                  },
                },
              ],
            },
          },

          // Centre: headline
          {
            type: 'div',
            props: {
              style: {
                display: 'flex',
                flexDirection: 'column',
                gap: '20px',
              },
              children: [
                {
                  type: 'div',
                  props: {
                    style: {
                      fontSize: '72px',
                      fontWeight: '800',
                      color: '#fafafa',
                      lineHeight: '1.1',
                      letterSpacing: '-0.03em',
                    },
                    children: 'Massachusetts\nProperty Auctions',
                  },
                },
                {
                  type: 'div',
                  props: {
                    style: {
                      fontSize: '28px',
                      fontWeight: '400',
                      color: '#71717a',
                      lineHeight: '1.4',
                    },
                    children: 'Live dates · Deposits · Auctioneer details · Updated daily',
                  },
                },
              ],
            },
          },

          // Bottom row: accent bar + tagline
          {
            type: 'div',
            props: {
              style: {
                display: 'flex',
                alignItems: 'center',
                gap: '16px',
              },
              children: [
                {
                  type: 'div',
                  props: {
                    style: {
                      width: '48px',
                      height: '4px',
                      background: 'hsl(158, 60%, 52%)',
                      borderRadius: '9999px',
                    },
                    children: '',
                  },
                },
                {
                  type: 'div',
                  props: {
                    style: {
                      fontSize: '18px',
                      fontWeight: '500',
                      color: 'hsl(158, 60%, 52%)',
                      letterSpacing: '0.02em',
                    },
                    children: 'auctionandcompany.com',
                  },
                },
              ],
            },
          },
        ],
      },
    } as any),
    {
      width: 1200,
      height: 630,
      fonts: [
        {
          name: 'Inter',
          data: font,
          weight: 400,
          style: 'normal',
        },
        {
          name: 'Inter',
          data: font,
          weight: 800,
          style: 'normal',
        },
      ],
    }
  );

  const resvg = new Resvg(svg, {
    fitTo: { mode: 'width', value: 1200 },
  });
  const png = resvg.render().asPng();

  return new Response(png, {
    headers: {
      'Content-Type': 'image/png',
      'Cache-Control': 'public, max-age=86400, s-maxage=86400',
    },
  });
};
