import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import axios from 'axios'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { login } from '@/features/auth/api/auth'
import { useAuthStore } from '@/features/auth/stores/auth-store'
import type { LoginResponse } from '@/features/auth/types'

const loginSchema = z.object({
  email: z
    .string()
    .min(1, '请输入邮箱')
    .email('邮箱格式不正确')
    .max(255, '邮箱长度过长'),
  password: z
    .string()
    .min(8, '密码至少需要 8 位')
    .max(128, '密码长度最大 128 位'),
})

type LoginFormValues = z.infer<typeof loginSchema>

export function LoginForm() {
  const navigate = useNavigate()
  const location = useLocation()
  const setAuth = useAuthStore((state) => state.setAuth)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: '',
      password: '',
    },
  })

  const onSubmit = async (values: LoginFormValues) => {
    setErrorMessage(null)
    try {
      const result: LoginResponse = await login(values)
      setAuth(result)
      const redirectTo = (location.state as { from?: string } | null)?.from ?? '/prompts'
      navigate(redirectTo, { replace: true })
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const message =
          (error.response?.data as { message?: string } | undefined)?.message ?? '登录失败，请稍后重试'
        setErrorMessage(message)
      } else {
        setErrorMessage('登录失败，请检查网络或稍后再试')
      }
    }
  }

  return (
    <form className="space-y-4" onSubmit={handleSubmit(onSubmit)}>
      {errorMessage ? <Alert variant="error">{errorMessage}</Alert> : null}

      <div className="space-y-2">
        <label className="block text-sm font-medium text-slate-700" htmlFor="email">
          邮箱
        </label>
        <Input id="email" type="email" placeholder="you@example.com" {...register('email')} />
        {errors.email ? (
          <p className="text-xs text-red-600">{errors.email.message}</p>
        ) : null}
      </div>

      <div className="space-y-2">
        <label className="block text-sm font-medium text-slate-700" htmlFor="password">
          密码
        </label>
        <Input id="password" type="password" placeholder="请输入密码" {...register('password')} />
        {errors.password ? (
          <p className="text-xs text-red-600">{errors.password.message}</p>
        ) : null}
      </div>

      <Button type="submit" className="w-full" disabled={isSubmitting}>
        {isSubmitting ? '登录中...' : '登录'}
      </Button>
    </form>
  )
}
