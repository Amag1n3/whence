import { useEffect, useState } from 'react'
import { Check, Copy } from 'lucide-react'
import { toast } from 'sonner'

/* Lifted out of App when /install and /docs needed it. Same rule as Chrome:
   three pages render code blocks now, and a block that exists three times
   drifts.

   The whole block is the click target (opencode's pattern): the corner icon
   is a hint, not the hit area. The container is a div with a click handler
   rather than a <button> wrapping everything, because <pre> and <div> are
   flow content and a <button> may only contain phrasing content — wrapping
   the block in one renders fine and is invalid, which is the kind of thing
   that breaks in a screen reader and nowhere else. The real <button> in the
   corner is what keyboard and AT users get; the container click is a
   mouse-only enhancement on top of it. */

export function Code({ children, caption }: { children: string; caption?: string }) {
  const [copied, setCopied] = useState(false)

  /* Revert the icon, and clear the timer if the block unmounts first —
     otherwise a page navigation mid-countdown sets state on a dead component. */
  useEffect(() => {
    if (!copied) return
    const t = window.setTimeout(() => setCopied(false), 2000)
    return () => window.clearTimeout(t)
  }, [copied])

  const copy = async (force = false) => {
    /* A click that ends a text selection is someone copying one flag by hand,
       not asking for the whole block. Swallowing their selection and toasting
       "copied" would be actively wrong. The corner button forces past this,
       since clicking it is unambiguous. */
    if (!force && window.getSelection()?.toString()) return

    try {
      await navigator.clipboard.writeText(children)
      setCopied(true)
      toast('Copied to clipboard')
    } catch {
      /* writeText rejects without a secure context, and navigator.clipboard
         is undefined outright on some older browsers. Leaving the button
         un-flipped is the honest signal: a tick that lied would send someone
         off to paste nothing and debug the wrong thing. */
    }
  }

  return (
    <div
      onClick={() => copy()}
      className="lit group relative cursor-pointer overflow-hidden rounded-md border border-white/10 bg-terminal transition-colors hover:border-white/20 hover:bg-white/[0.02]"
    >
      {caption && (
        <div className="border-b border-white/[0.07] px-4 py-2 pr-12 font-mono text-[11px] text-dim">
          {caption}
        </div>
      )}

      {/* Hidden until hover on a pointer device, always visible on touch —
          where there is no hover to reveal it and an invisible affordance is
          just a missing one. Sits on the outer container, not inside the
          scroll area, so it stays put while a long line scrolls under it, and
          is opaque so it occludes the code rather than blending into it. */}
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation()
          copy(true)
        }}
        aria-label={copied ? 'Copied' : 'Copy to clipboard'}
        className="absolute top-2 right-2 z-10 inline-flex size-7 items-center justify-center border border-white/10 bg-terminal text-dim opacity-0 transition-all group-hover:opacity-100 hover:border-white/25 hover:text-silt focus-visible:opacity-100 max-sm:opacity-100"
      >
        {copied ? (
          <Check className="size-3.5 text-silt" strokeWidth={2.5} />
        ) : (
          <Copy className="size-3.5" />
        )}
      </button>
      {/* No aria-live region here on purpose: sonner renders its own, and
          two would announce the copy twice. */}

      {/* pr-12 reserves the button's corner, so a one-line command does not
          sit underneath it at rest. */}
      <div className="term-scroll overflow-x-auto px-4 py-3.5 pr-12">
        <pre className="min-w-max font-mono text-[12.5px] leading-[1.75] text-silt/85">
          {children}
        </pre>
      </div>
    </div>
  )
}
