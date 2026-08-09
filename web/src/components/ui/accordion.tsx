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
    /* forceMount keeps closed answers in the DOM, so Googlebot sees all 61
     * instead of 61 headings and the one answer that opens by default —
     * roughly 4,000 words of the site's largest page. Content hidden with CSS
     * is indexed; content that was never mounted is not there to index.
     *
     * The animation does NOT use Radix's keyframes, and cannot. Those
     * interpolate to --radix-accordion-content-height, which the collapsible
     * primitive measures with getBoundingClientRect. forceMount pins
     * `isPresent` true, so Radix never applies its own `hidden`, so we have to
     * hide it ourselves — and anything that hides it (display:none, height:0)
     * is also unmeasurable, leaving the keyframe interpolating to zero. The
     * animation was resting on a measurement that could not happen.
     *
     * grid-template-rows 0fr→1fr needs no measured value at all: the row
     * sizes to the content, and the browser interpolates between them. The
     * inner overflow-hidden wrapper is what makes that collapse rather than
     * overflow, and is not optional.
     *
     * `visibility` is in the transition list on purpose. It animates
     * discretely — flipping to `visible` at the START of the transition and
     * to `hidden` at the END — which is exactly the behaviour wanted here:
     * content appears immediately on open, and stays rendered through the
     * whole collapse before leaving the accessibility tree. Without it,
     * height-zero content would still be read aloud by a screen reader. */
    <AccordionPrimitive.Content
      forceMount
      data-slot="accordion-content"
      className="grid overflow-hidden transition-[grid-template-rows,visibility] duration-200 ease-out data-[state=closed]:invisible data-[state=closed]:grid-rows-[0fr] data-[state=open]:visible data-[state=open]:grid-rows-[1fr]"
      {...props}
    >
      <div className="overflow-hidden">
        <div
          className={cn(
            "max-w-[68ch] pt-0 pb-6 text-[15.5px] leading-[1.7] text-muted-foreground",
            className
          )}
        >
          {children}
        </div>
      </div>
    </AccordionPrimitive.Content>
  )
}

export { Accordion, AccordionItem, AccordionTrigger, AccordionContent }
