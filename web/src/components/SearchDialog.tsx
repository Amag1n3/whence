import { lazy, Suspense, useEffect, useState } from 'react'
import { Search } from 'lucide-react'

import { cn } from '@/lib/utils'

/* Site-wide search. Lives in the header, so it is on every page.
 *
 * This module is only the trigger: a button and a key handler, a few hundred
 * bytes. Everything expensive — cmdk, the Radix dialog, and the content index
 * that pulls in the entire FAQ — sits behind React.lazy in SearchPalette and
 * loads the first time someone opens it. A static import cost every page 19kB
 * gzipped for a feature most visitors never touch. */

const SearchPalette = lazy(() => import('@/components/SearchPalette'))

export function SearchDialog() {
  const [open, setOpen] = useState(false)
  /* Once mounted, the palette stays mounted — remounting on every close would
     re-run the lazy import's module resolution and throw away cmdk's state
     for no benefit. `open` drives visibility from then on. */
  const [mounted, setMounted] = useState(false)

  const show = () => {
    setMounted(true)
    setOpen(true)
  }

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setMounted(true)
        setOpen((v) => !v)
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  return (
    <>
      <button
        type="button"
        onClick={show}
        aria-label="Search the site"
        className={cn(
          'group inline-flex h-8 items-center gap-2 border border-white/10 bg-white/[0.02] text-dim transition-colors hover:border-white/25 hover:text-silt',
          // Reads as a search field where there is room, collapses to the icon
          // on a phone — the header already wraps to two rows at 360px.
          'w-8 justify-center sm:w-56 sm:justify-start sm:px-2.5',
        )}
      >
        <Search className="size-3.5 shrink-0" />
        <span className="hidden font-mono text-[12px] sm:inline">Search</span>
        <kbd className="ml-auto hidden border border-white/10 px-1.5 py-0.5 font-mono text-[10px] text-dim sm:inline">
          ⌘K
        </kbd>
      </button>

      {mounted && (
        <Suspense fallback={null}>
          <SearchPalette open={open} onOpenChange={setOpen} />
        </Suspense>
      )}
    </>
  )
}
