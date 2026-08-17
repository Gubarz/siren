import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config.js'

export default mergeConfig(
  viteConfig,
  defineConfig({
    resolve: {
      conditions: ['browser'],
    },
    test: {
      // Component tests opt into jsdom per-file via `// @vitest-environment jsdom`.
      setupFiles: ['./test/setup.js'],
      server: {
        deps: {
          inline: ['@testing-library/svelte', '@testing-library/svelte-core'],
        },
      },
    },
  }),
)
