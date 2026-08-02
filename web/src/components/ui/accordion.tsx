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
          "group flex flex-1 items-start justify-between gap-6 py-5 text-left text-[15.5px] leading-[1.55] font-medium text-silt transition-colors outline-none hover:text-ochre focus-visible:text-ochre disabled:pointer-events-none disabled:opacity-50",
          className
        )}
        {...props}
      >
        {children}
        <Plus
          className="mt-0.5 size-4 shrink-0 text-dim transition-transform duration-200 group-hover:text-ochre group-data-[state=open]:rotate-45"
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
    <AccordionPrimitive.Content
      data-slot="accordion-content"
      className="overflow-hidden data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down"
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
