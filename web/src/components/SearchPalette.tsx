import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { INDEX } from '@/content/search-index'

/* The palette itself, in its own module so React.lazy can keep it — and cmdk,
   the Radix dialog and the whole content index — out of every page's bundle.
   Mounted only once the reader opens search, which is why the index can be a
   plain static import here.
 *
 * Splitting this out took the shared chunk back from 403kB to 343kB. Most
 * visitors never open search, and they should not pay for it. */

/** Every term must appear. Returns 0 to hide, higher is a better hit.
 *
 *  cmdk's default scorer is a fuzzy matcher tuned for command names: it will
 *  happily match "gate" against "Get a non-empty store" through scattered
 *  letters, which is right for a palette of commands and wrong for prose.
 *  This requires whole substrings and weights the title above the body, so
 *  "orphan" ranks the question about orphaned records above the several
 *  answers that mention orphaning in passing. */
function score(value: string, search: string, keywords?: string[]): number {
  const q = search.toLowerCase().trim()
  if (!q) return 1

  const title = value.toLowerCase()
  const body = (keywords?.join(' ') ?? '').toLowerCase()
  const terms = q.split(/\s+/).filter(Boolean)

  let total = 0
  for (const term of terms) {
    const inTitle = title.includes(term)
    const inBody = body.includes(term)
    // AND, not OR. A second word should narrow the results, not widen them.
    if (!inTitle && !inBody) return 0
    total += inTitle ? (title.startsWith(term) ? 6 : 4) : 1
  }

  // A whole-phrase hit in the title beats the same words scattered about.
  if (title.includes(q)) total += 4

  // Normalised, so a long query cannot out-score a better short one.
  return total / terms.length
}

const GROUPS = ['Pages', 'Commands', 'Questions']

export default function SearchPalette({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  return (
    <CommandDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Search"
      description="Search pages, commands and questions"
      commandProps={{ filter: score }}
    >
      <CommandInput placeholder="Search pages, commands and questions…" />
      <CommandList>
        <CommandEmpty>
          <span className="font-mono text-[12.5px]">No matches.</span>
        </CommandEmpty>
        {GROUPS.map((group) => {
          const hits = INDEX.filter((h) => h.group === group)
          if (!hits.length) return null
          return (
            <CommandGroup key={group} heading={group}>
              {hits.map((hit) => (
                <CommandItem
                  key={`${hit.href}-${hit.title}`}
                  value={hit.title}
                  keywords={[hit.body]}
                  onSelect={() => {
                    window.location.href = hit.href
                  }}
                  className="flex-col items-start gap-1"
                >
                  <span className="text-[14px] leading-[1.4] text-silt">{hit.title}</span>
                  {hit.body && (
                    <span className="line-clamp-1 text-[12.5px] leading-[1.4] text-dim">
                      {hit.body}
                    </span>
                  )}
                </CommandItem>
              ))}
            </CommandGroup>
          )
        })}
      </CommandList>
    </CommandDialog>
  )
}
