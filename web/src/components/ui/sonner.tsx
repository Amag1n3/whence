import { Toaster as Sonner, type ToasterProps } from "sonner"

/* LOCAL EDIT to a registry file. Two things were stripped from what
   `shadcn add sonner` generates:

   1. `next-themes`. The generated wrapper reads the active theme from it,
      which this site does not have — it is dark-only, set by `class="dark"`
      on <html>. The dependency came in with the install and went straight
      back out; `theme` is hardcoded instead.
   2. The success/info/warning/error icon map. Nothing here calls those
      variants, and five unused lucide imports is five unused lucide imports.

   Text is mono at the documented `code` step: a toast is interface chrome,
   and every other piece of interface text on this site is mono.
   Re-running `shadcn add sonner` reverts all of this. */

const Toaster = ({ ...props }: ToasterProps) => {
  return (
    <Sonner
      theme="dark"
      className="toaster group"
      toastOptions={{
        classNames: {
          toast: 'font-mono text-[12.5px] tracking-[-0.01em]',
        },
      }}
      style={
        {
          '--normal-bg': 'var(--popover)',
          '--normal-text': 'var(--popover-foreground)',
          '--normal-border': 'var(--border)',
          '--border-radius': 'var(--radius)',
        } as React.CSSProperties
      }
      {...props}
    />
  )
}

export { Toaster }
