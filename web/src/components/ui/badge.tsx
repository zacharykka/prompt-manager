import { clsx } from 'clsx'
import type { PropsWithChildren } from 'react'

type BadgeVariant = 'default' | 'neutral'

interface BadgeProps extends PropsWithChildren {
  variant?: BadgeVariant
  className?: string
}

const variantClasses: Record<BadgeVariant, string> = {
  default:
    'bg-brand-50 text-brand-700 ring-1 ring-inset ring-brand-200 dark:bg-brand-600/10 dark:text-brand-200',
  neutral:
    'bg-slate-100 text-slate-700 ring-1 ring-inset ring-slate-200 dark:bg-slate-700/30 dark:text-slate-200',
}

export function Badge({ variant = 'default', className, children }: BadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium',
        variantClasses[variant],
        className,
      )}
    >
      {children}
    </span>
  )
}
