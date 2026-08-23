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
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      /* React is swapped for Preact's compat layer at build time. react and
         react-dom were ~200K of the ~215K every page had to parse before
         anything appeared; compat is ~25K, and the shared chunk went 213K to
         35K (68.2K to 12.3K gzipped).

         `react` and `react-dom` stay in package.json on purpose. Nothing
         bundles them — this alias replaces them — but @types/react is what
         typecheck runs against, so tsc still validates every call against the
         real React API and would flag anything compat does not implement.
         Removing them would silently drop that check.

         compat reports itself as React 18.3.1. Anything added later that
         branches on the React version, or relies on a 19-only behaviour like
         ref-as-a-prop on a plain function component, will see 18 and needs
         checking in a browser rather than trusting the build. */
      react: 'preact/compat',
      'react-dom': 'preact/compat',
      'react-dom/client': 'preact/compat/client',
      'react/jsx-runtime': 'preact/jsx-runtime',
    },
  },
  build: {
    // Seven entries, not a router. The site is static documents; a
    // client-side router would be a dependency and a runtime for something the
    // platform already does, and it would cost each page its own title,
    // description and canonical URL — which are the point of giving it a page
    // at all. /install in particular is the URL people paste at each other.
    rollupOptions: {
      input: {
        index: fileURLToPath(new URL('./index.html', import.meta.url)),
        why: fileURLToPath(new URL('./why.html', import.meta.url)),
        contact: fileURLToPath(new URL('./contact.html', import.meta.url)),
        install: fileURLToPath(new URL('./install.html', import.meta.url)),
        docs: fileURLToPath(new URL('./docs.html', import.meta.url)),
        faq: fileURLToPath(new URL('./faq.html', import.meta.url)),
        notes: fileURLToPath(new URL('./notes.html', import.meta.url)),
        trials: fileURLToPath(new URL('./trials.html', import.meta.url)),
        privacy: fileURLToPath(new URL('./privacy.html', import.meta.url)),
      },
    },
  },
})
