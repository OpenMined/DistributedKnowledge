import { defineConfig, externalizeDepsPlugin } from 'electron-vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { resolve } from 'path'

export default defineConfig({
  main: {
    plugins: [externalizeDepsPlugin()],
    resolve: {
      alias: {
        '@shared': resolve('src/shared'),
        '@main': resolve('src/main'),
        '@services': resolve('src/main/services'),
        '@ipc': resolve('src/main/ipc'),
        '@utils': resolve('src/main/utils'),
        '@windows': resolve('src/main/windows')
      }
    }
  },
  preload: {
    plugins: [externalizeDepsPlugin()],
    resolve: {
      alias: {
        '@shared': resolve('src/shared'),
        '@main': resolve('src/main'),
        '@preload': resolve('src/preload'),
        '@apis': resolve('src/preload/apis')
      }
    }
  },
  renderer: {
    plugins: [svelte()],
    resolve: {
      alias: {
        '@': resolve('src/renderer/src'),
        '@renderer': resolve('src/renderer/src'),
        '@components': resolve('src/renderer/src/components'),
        '@lib': resolve('src/renderer/src/lib'),
        '@assets': resolve('src/renderer/src/assets'),
        '@shared': resolve('src/shared')
      }
    },
    build: {
      outDir: 'out/renderer',
      emptyOutDir: true,
      rollupOptions: {
        input: {
          index: resolve('src/renderer/index.html')
        }
      },
      assetsInlineLimit: 0, // Prevent inlining assets
      assetsDir: 'assets' // Output directory for assets
    },
    publicDir: resolve('src/renderer/public') // Specify public directory explicitly
  }
})
