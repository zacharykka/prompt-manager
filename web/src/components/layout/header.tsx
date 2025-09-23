import { Link, useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { useAuthStore } from '@/features/auth/stores/auth-store'

interface HeaderProps {
  userEmail?: string
}

export function Header({ userEmail }: HeaderProps) {
  const navigate = useNavigate()
  const clearAuth = useAuthStore((state) => state.clearAuth)

  const handleSignOut = () => {
    clearAuth()
    navigate('/auth/login', { replace: true })
  }

  return (
    <header className="border-b border-slate-200 bg-white">
      <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between px-6">
        <Link to="/prompts" className="text-lg font-semibold text-slate-900">
          Prompt Manager
        </Link>
        <div className="flex items-center gap-4 text-sm text-slate-600">
          {userEmail ? <span>{userEmail}</span> : null}
          <Button variant="secondary" size="sm" onClick={handleSignOut}>
            退出登录
          </Button>
        </div>
      </div>
    </header>
  )
}
