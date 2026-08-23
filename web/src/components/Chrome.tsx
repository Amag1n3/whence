import { ArrowUpRight } from 'lucide-react'
import type { MouseEvent } from 'react'

import { SearchDialog } from '@/components/SearchDialog'
import { cn } from '@/lib/utils'

export const REPO = 'https://github.com/Amag1n3/whence'
export const EMAIL = 'amogh@whence.fyi'
export const SHELL = 'mx-auto w-full max-w-[1280px] px-6 sm:px-10'

/* Header and footer live here rather than in App because there are four pages
   now, and chrome that exists four times drifts. The landing page scrolls to its
   own top; every other page navigates home — so the logo takes a handler rather
   than assuming. */

/** One list, rendered by both header and footer, so a page added to one cannot
 *  go missing from the other. That drift is exactly why this file exists. */
const NAV: { href: string; label: string; key: PageKey }[] = [
  { href: '/why', label: 'why', key: 'why' },
  { href: '/notes', label: 'notes', key: 'notes' },
  { href: '/trials', label: 'trials', key: 'trials' },
  { href: '/install', label: 'install', key: 'install' },
  { href: '/docs', label: 'docs', key: 'docs' },
  { href: '/faq', label: 'questions', key: 'faq' },
]

/* 'contact' and 'privacy' are PageKeys but deliberately not NAV entries. The
   header is at five items plus the logo already and wraps to two rows on a
   narrow phone; both are destinations people look for in a footer, not
   something to spend header width on. They are rendered explicitly by Footer
   below. */
export type PageKey =
  | 'why'
  | 'notes'
  | 'trials'
  | 'install'
  | 'docs'
  | 'faq'
  | 'contact'
  | 'privacy'

export function Header({
  onLogo,
  railed = false,
  current,
}: {
  onLogo?: (e: MouseEvent<HTMLAnchorElement>) => void
  /** Offset for the strata rail, which only the landing page has. */
  railed?: boolean
  /** Which page is being rendered, for aria-current. Absent on the landing page. */
  current?: PageKey
}) {
  return (
    <header
      className={cn(
        'fixed inset-x-0 top-0 z-50 border-b border-white/[0.06] bg-basin/75 backdrop-blur-xl',
        railed && 'lg:pl-14',
      )}
    >
      <div className={cn(SHELL, 'flex h-14 items-center gap-6')}>
        <a
          href={onLogo ? '#surface' : '/'}
          onClick={onLogo}
          className="font-mono text-[13.5px] font-medium tracking-tight"
        >
          <span className="text-ochre">●</span> whence
        </a>
        {/* min-h-6 on every link, not padding on the row: WCAG 2.2 SC 2.5.8 wants
            a 24px target, and a 12.5px text node gives about 16. These are nav
            items in a row rather than links inside a sentence, so the inline
            exception does not cover them.
            gap-x-5 with a wrap allowance: four items plus the logo overflow a
            360px viewport at gap-6, and a nav that clips is worse than one that
            takes two rows. */}
        <div className="ml-auto">
          <SearchDialog />
        </div>
        <nav className="flex flex-wrap items-center justify-end gap-x-5 gap-y-1 font-mono text-[12.5px] text-muted-foreground">
          {NAV.map((item) => (
            <a
              key={item.key}
              href={item.href}
              className={cn(
                'inline-flex min-h-6 items-center transition-colors hover:text-silt',
                current === item.key && 'text-silt',
              )}
              aria-current={current === item.key ? 'page' : undefined}
            >
              {item.label}
            </a>
          ))}
          <a
            href={REPO}
            className="inline-flex min-h-6 items-center gap-0.5 transition-colors hover:text-silt"
          >
            github <ArrowUpRight className="size-3.5" />
          </a>
        </nav>
      </div>
    </header>
  )
}

export function Footer() {
  return (
    <footer className="border-t border-white/[0.07]">
      <div
        className={cn(
          SHELL,
          'flex flex-wrap items-center gap-x-7 gap-y-2 py-9 font-mono text-[12.5px] text-dim',
        )}
      >
        <span className="text-silt/70">
          <span className="text-ochre">●</span> whence
        </span>
        {NAV.map((item) => (
          <a
            key={item.key}
            href={item.href}
            className="inline-flex min-h-6 items-center transition-colors hover:text-silt"
          >
            {item.label}
          </a>
        ))}
        <a
          href={REPO}
          className="inline-flex min-h-6 items-center transition-colors hover:text-silt"
        >
          github
        </a>
        <a
          href="/contact"
          className="inline-flex min-h-6 items-center transition-colors hover:text-silt"
        >
          contact
        </a>
        <a
          href="/privacy"
          className="inline-flex min-h-6 items-center transition-colors hover:text-silt"
        >
          privacy
        </a>
        <span className="sm:ml-auto">started 2026-07-31</span>
      </div>
    </footer>
  )
}
