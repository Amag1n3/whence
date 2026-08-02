/* Lifted out of App when /install and /docs needed it. Same rule as Chrome:
   three pages render code blocks now, and a block that exists three times
   drifts. Verbatim from the landing page — rounded-md rather than the xl the
   design file names for terminal surfaces, because that is what shipped. */

export function Code({ children, caption }: { children: string; caption?: string }) {
  return (
    <div className="lit overflow-hidden rounded-md border border-white/10 bg-terminal">
      {caption && (
        <div className="border-b border-white/[0.07] px-4 py-2 font-mono text-[11px] text-dim">
          {caption}
        </div>
      )}
      <div className="term-scroll overflow-x-auto px-4 py-3.5">
        <pre className="min-w-max font-mono text-[12.5px] leading-[1.75] text-silt/85">
          {children}
        </pre>
      </div>
    </div>
  )
}
