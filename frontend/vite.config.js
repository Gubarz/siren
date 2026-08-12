import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import tailwind from '@tailwindcss/vite'
import path from 'path'

export default defineConfig({
  plugins: [tailwind(), svelte()],
  // `wails3 dev` hands the dev-server port to both the app
  // (FRONTEND_DEVSERVER_URL) and vite (WAILS_VITE_PORT).
  server: {
    port: Number(process.env.WAILS_VITE_PORT) || 5173,
    strictPort: true,
  },
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
