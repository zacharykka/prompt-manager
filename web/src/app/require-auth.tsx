import type { PropsWithChildren } from 'react'
import { Navigate, useLocation } from 'react-router-dom'

import { useAuthStore } from '@/features/auth/stores/auth-store'

export function RequireAuth({ children }: PropsWithChildren) {
  const location = useLocation()
  const hasAccessToken = useAuthStore((state) => Boolean(state.tokens?.accessToken))

  if (!hasAccessToken) {
    const from = `${location.pathname}${location.search}${location.hash}`
    return (
      <Navigate
        to="/auth/login"
        replace
        state={{ from }}
      />
    )
  }

  return children
}
