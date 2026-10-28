import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { lingui } from '@lingui/vite-plugin'
import path from 'node:path'

export default defineConfig({
  build: {
    // Keep the Vite 6 output targets when upgrading the bundler.
    target: ['es2020', 'edge88', 'firefox78', 'chrome87', 'safari14'],
  },
  resolve: {
    alias: { '@': path.resolve(import.meta.dirname, './src') },
  },
  plugins: [
    react(),
    tailwindcss(),
    lingui({ macroTransform: true }),
  ],
})
