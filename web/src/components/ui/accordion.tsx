import * as React from "react"
import { Accordion as AccordionPrimitive } from "radix-ui"
import { Plus } from "lucide-react"

import { cn } from "@/lib/utils"

function Accordion({
  ...props
}: React.ComponentProps<typeof AccordionPrimitive.Root>) {
  return <AccordionPrimitive.Root data-slot="accordion" {...props} />
}

function AccordionItem({
  className,
  ...props
}: React.ComponentProps<typeof AccordionPrimitive.Item>) {
  return (
    <AccordionPrimitive.Item
      data-slot="accordion-item"
      className={cn("border-b border-white/[0.07] last:border-b-0", className)}
      {...props}
    />
  )
}

/* The marker rotates into a minus rather than swapping glyph, so the control
   reads as one object in two states. Ochre because it is an accent on an
   identity element — verdigris and cinnabar carry meaning here and would be
   claiming something about the answer. */
function AccordionTrigger({
  className,
  children,
  ...props
}: React.ComponentProps<typeof AccordionPrimitive.Trigger>) {
  return (
    <AccordionPrimitive.Header className="flex">
      <AccordionPrimitive.Trigger
        data-slot="accordion-trigger"
        className={cn(
          /* Hover is a background lift, not a colour change. It used to be
             hover:text-ochre from a text-silt base — which the monochrome
             repaint turned into white-on-white, an invisible hover state.
             A background tint does not depend on there being a second hue.
             The negative margin plus matching padding lets the tint bleed
             past the text column so the whole row lights up, the way a list
             row should. */
          "group -mx-4 flex flex-1 items-start justify-between gap-6 px-4 py-5 text-left text-[15.5px] leading-[1.55] font-medium text-silt transition-colors outline-none hover:bg-white/[0.035] focus-visible:bg-white/[0.035] disabled:pointer-events-none disabled:opacity-50",
          className
        )}
        {...props}
      >
        {children}
        <Plus
          className="mt-0.5 size-4 shrink-0 text-dim transition-transform duration-200 group-hover:text-silt group-data-[state=open]:rotate-45"
          strokeWidth={2}
          aria-hidden
        />
      </AccordionPrimitive.Trigger>
    </AccordionPrimitive.Header>
  )
}

function AccordionContent({
  className,
  children,
  ...props
}: React.ComponentProps<typeof AccordionPrimitive.Content>) {
  return (
    /* forceMount keeps closed answers in the DOM.
     *
     * Radix unmounts closed content by default, which meant Googlebot
     * rendered /faq and saw 61 headings plus the single answer that opens by
     * default — roughly 4,000 words of the site's largest page did not exist
     * as far as search was concerned. Content that is in the DOM but hidden
     * with CSS is indexed normally; content that was never mounted is not
     * there to index.
     *
     * The cost is the close animation: `hidden` applies the moment state
     * flips, so closing snaps rather than collapsing. Opening still animates.
     * display:none also keeps the closed text out of the accessibility tree,
     * which height:0 would not have done — a screen reader would have read
     * all 61 answers straight through. Correctness over the animation. */
    <AccordionPrimitive.Content
      forceMount
      data-slot="accordion-content"
      className="overflow-hidden data-[state=closed]:hidden data-[state=open]:animate-accordion-down"
      {...props}
    >
      <div
        className={cn(
          "max-w-[68ch] pt-0 pb-6 text-[15.5px] leading-[1.7] text-muted-foreground",
          className
        )}
      >
        {children}
      </div>
    </AccordionPrimitive.Content>
  )
}

export { Accordion, AccordionItem, AccordionTrigger, AccordionContent }
