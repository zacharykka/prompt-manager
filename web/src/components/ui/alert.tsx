import { clsx } from 'clsx'
import type { PropsWithChildren } from 'react'

type AlertVariant = 'info' | 'error' | 'success'

interface AlertProps extends PropsWithChildren {
  variant?: AlertVariant
  className?: string
}

const variantClasses: Record<AlertVariant, string> = {
  info: 'bg-slate-100 text-slate-700 border-slate-200',
  error: 'bg-red-50 text-red-700 border-red-200',
  success: 'bg-emerald-50 text-emerald-700 border-emerald-200',
}

export function Alert({ variant = 'info', className, children }: AlertProps) {
  return (
    <div
      className={clsx(
        'rounded-md border px-4 py-3 text-sm',
        variantClasses[variant],
        className,
      )}
    >
      {children}
    </div>
  )
}
