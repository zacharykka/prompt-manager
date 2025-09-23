import { Outlet } from 'react-router-dom'

import { Header } from '@/components/layout/header'
import { useAuthStore } from '@/features/auth/stores/auth-store'

export function RootLayout() {
  const user = useAuthStore((state) => state.user)

  return (
    <div className="min-h-screen bg-slate-100">
      <Header userEmail={user?.email ?? ''} />
      <main className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-6 py-10">
        <Outlet />
      </main>
    </div>
  )
}
