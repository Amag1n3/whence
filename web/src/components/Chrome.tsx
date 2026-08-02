import { ArrowUpRight } from 'lucide-react'
import type { MouseEvent } from 'react'

import { cn } from '@/lib/utils'

export const REPO = 'https://github.com/Amag1n3/whence'
export const SHELL = 'mx-auto w-full max-w-[1280px] px-6 sm:px-10'

/* Header and footer live here rather than in App because there are two pages
   now, and chrome that exists twice drifts. The landing page scrolls to its own
   top; /faq navigates home — so the logo takes a handler rather than assuming. */

export function Header({
  onLogo,
  railed = false,
}: {
  onLogo?: (e: MouseEvent<HTMLAnchorElement>) => void
  /** Offset for the strata rail, which only the landing page has. */
  railed?: boolean
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
            exception does not cover them. */}
        <nav className="ml-auto flex items-center gap-6 font-mono text-[12.5px] text-muted-foreground">
          <a
            href={onLogo ? '/faq' : '/'}
            className="inline-flex min-h-6 items-center transition-colors hover:text-silt"
            aria-current={onLogo ? undefined : 'page'}
          >
            {onLogo ? 'questions' : 'home'}
          </a>
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
        <a
          href={REPO}
          className="inline-flex min-h-6 items-center transition-colors hover:text-silt"
        >
          github
        </a>
        <a
          href="/faq"
          className="inline-flex min-h-6 items-center transition-colors hover:text-silt"
        >
          questions
        </a>
        <a
          href="mailto:amogh@whence.fyi"
          className="inline-flex min-h-6 items-center transition-colors hover:text-silt"
        >
          amogh@whence.fyi
        </a>
        <span className="sm:ml-auto">started 2026-07-31</span>
      </div>
    </footer>
  )
}
