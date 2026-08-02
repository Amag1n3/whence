import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath } from 'node:url'

export default defineConfig({
  // Relative asset paths, so dist/index.html opens straight off the filesystem
  // for review. Harmless once Pages serves it from the root, and it still
  // resolves from /faq because that URL has no trailing slash — Pages serves
  // faq.html there and normalises /faq/ back to it.
  base: './',
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  build: {
    // Two entries, not a router. The site is two static documents; a
    // client-side router would be a dependency and a runtime for something the
    // platform already does, and it would cost /faq its own title, description
    // and canonical URL — which are the point of giving it a page at all.
    rollupOptions: {
      input: {
        index: fileURLToPath(new URL('./index.html', import.meta.url)),
        faq: fileURLToPath(new URL('./faq.html', import.meta.url)),
      },
    },
  },
})
