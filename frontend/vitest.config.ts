import { transform as compileAstro } from '@astrojs/compiler';
import { transformWithEsbuild } from 'vite';
import { defineConfig } from 'vitest/config';

const astroCompilerRuntimeId = '\0astro-test-compiler-runtime';
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
      },
      load(id) {
        if (id !== astroCompilerRuntimeId) return;
        return [
          "export * from 'astro/runtime/server/index.js';",
          'export const createMetadata = (_filename, metadata) => metadata;',
        ].join('\n');
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
