import type { HTMLAttributes } from 'react'

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  padded?: boolean
}

export function Card({ padded = true, className = '', children, ...rest }: CardProps) {
  return (
    <div
      className={`rounded-xl border border-border bg-surface-raised shadow-sm ${padded ? 'p-4' : ''} ${className}`}
      {...rest}
    >
      {children}
    </div>
  )
}
