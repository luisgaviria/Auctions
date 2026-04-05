import { defineConfig, fontProviders } from 'astro/config';
import react from '@astrojs/react';
import svelte from '@astrojs/svelte';
import vercel from '@astrojs/vercel';

export default defineConfig({
  adapter: vercel(),
  output: 'server',
  integrations: [react(), svelte()],
  build: {
    // Inline all CSS into <style> tags in the HTML, eliminating the two
    // separate CSS requests (~3.2 KiB + ~3.4 KiB) from the critical path.
    inlineStylesheets: 'always',
  },
  vite: {
    envPrefix: 'VITE_',
  },
  // Astro 6 Fonts API — self-optimised, no manual <link> tags required
  fonts: [
    {
      provider: fontProviders.fontshare(),
      name: 'Satoshi',
      cssVariable: '--font-sans',
      weights: [400, 500, 700],
    },
    {
      provider: fontProviders.fontsource(),
      name: 'JetBrains Mono',
      cssVariable: '--font-mono',
      weights: [400, 500],
    },
  ],
});
