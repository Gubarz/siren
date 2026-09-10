import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import tailwind from '@tailwindcss/vite'
import fs from 'fs'
import path from 'path'

// main.go embeds the built frontend with `//go:embed all:frontend/dist`, which
// needs that directory to exist and hold at least one file, so dist/.gitkeep is
// tracked to keep `go build` working on a fresh clone. Vite empties outDir on
// every build and takes the placeholder with it, which leaves the working tree
// dirty and breaks the next Go build until someone restores it by hand. Write it
// back once the output is on disk. outDir stays emptied, so no stale assets from
// a previous build can end up embedded in the binary.
const restoreDistPlaceholder = () => {
  let placeholder = 'dist/.gitkeep'
  return {
    name: 'siren:restore-dist-placeholder',
    configResolved(config) {
      placeholder = path.resolve(config.root, config.build.outDir, '.gitkeep')
    },
    closeBundle() {
      fs.mkdirSync(path.dirname(placeholder), {recursive: true})
      fs.writeFileSync(placeholder, '')
    },
  }
}

export default defineConfig({
  plugins: [restoreDistPlaceholder(), tailwind(), svelte()],
  // `wails3 dev` hands the dev-server port to both the app
  // (FRONTEND_DEVSERVER_URL) and vite (WAILS_VITE_PORT).
  server: {
    // wails3's dev proxy dials the dev server over IPv4 (127.0.0.1). On hosts
    // where localhost resolves to ::1 first, vite would bind IPv6-only and the
    // proxy gets ECONNREFUSED — pin IPv4 loopback so `wails3 dev` works.
    host: '127.0.0.1',
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
