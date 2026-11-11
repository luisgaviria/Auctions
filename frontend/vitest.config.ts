import { transform as compileAstro } from '@astrojs/compiler';
import { transformWithEsbuild } from 'vite';
import { defineConfig } from 'vitest/config';

const astroCompilerRuntimeId = '\0astro-test-compiler-runtime';
const astroTestFontId = '\0astro-test-font';
const astroTestTransitionsId = '\0astro-test-transitions';
type AstroTestTransformOptions = NonNullable<Parameters<typeof compileAstro>[1]> & {
  mode: 'server';
};

export default defineConfig({
  plugins: [
    {
      name: 'astro-test-transform',
      enforce: 'pre',
      resolveId(id) {
        if (id === 'astro/compiler-runtime') return astroCompilerRuntimeId;
        if (id === 'astro/components/Font.astro') return astroTestFontId;
        if (id === 'astro:transitions') return astroTestTransitionsId;
      },
      load(id) {
        if (id === astroCompilerRuntimeId) {
          return [
            "export * from 'astro/runtime/server/index.js';",
            'export const createMetadata = (_filename, metadata) => metadata;',
          ].join('\n');
        }
        if (id === astroTestFontId) {
          return [
            "import { createComponent } from 'astro/runtime/server/index.js';",
            'export default createComponent(() => "");',
          ].join('\n');
        }
        if (id === astroTestTransitionsId) {
          return [
            "import { createComponent } from 'astro/runtime/server/index.js';",
            'export const ClientRouter = createComponent(() => "");',
          ].join('\n');
        }
      },
      async transform(source, id) {
        if (!id.endsWith('.astro')) return;

        const compilerOptions: AstroTestTransformOptions = {
          filename: id,
          internalURL: 'astro/compiler-runtime',
          mode: 'server',
        };
        const compiled = await compileAstro(source, compilerOptions);
        return transformWithEsbuild(compiled.code, `${id}.ts`, { loader: 'ts' });
      },
    },
  ],
});
