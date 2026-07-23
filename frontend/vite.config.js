import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import tailwind from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [tailwind(), svelte()],
  resolve: {
    alias: {
      $components: path.resolve('src/lib/components'),
      $features: path.resolve('src/lib/features'),
      $stores: path.resolve('src/lib/stores'),
      $utils: path.resolve('src/lib/utils'),
      $api: path.resolve('src/lib/api'),
    },
  },
})
