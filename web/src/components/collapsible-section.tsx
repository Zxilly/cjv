import type { ReactNode } from 'react'
import { ChevronRight } from 'lucide-react'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from './ui/collapsible'

interface CollapsibleSectionProps {
  title: string
  initial?: boolean
  children: ReactNode
}

export function CollapsibleSection({ title, initial = false, children }: CollapsibleSectionProps) {
  return (
    <Collapsible defaultOpen={initial} className="px-6 py-4">
      <CollapsibleTrigger className="group cursor-pointer text-base text-cj dark:text-cj-light select-none flex items-center gap-1 w-full text-left">
        <ChevronRight className="size-4 transition-transform duration-200 group-data-[state=open]:rotate-90" />
        <span className="group-hover:underline">{title}</span>
      </CollapsibleTrigger>
      <CollapsibleContent className="overflow-hidden data-[state=open]:animate-collapsible-down data-[state=closed]:animate-collapsible-up">
        {children}
      </CollapsibleContent>
    </Collapsible>
  )
}
