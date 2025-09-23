import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

import { LoginForm } from '@/features/auth/components/login-form'
import { useAuthStore } from '@/features/auth/stores/auth-store'

export function LoginPage() {
  const navigate = useNavigate()
  const isAuthenticated = useAuthStore((state) =>
    Boolean(state.tokens?.accessToken),
  )

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/prompts', { replace: true })
    }
  }, [isAuthenticated, navigate])

  return (
    <div className="space-y-6">
      <div className="space-y-2 text-center">
        <h1 className="text-2xl font-semibold text-slate-900">欢迎登录 Prompt Manager</h1>
        <p className="text-sm text-slate-600">
          使用已注册的邮箱与密码登录系统，管理您的 Prompt 资源。
        </p>
      </div>
      <LoginForm />
    </div>
  )
}
