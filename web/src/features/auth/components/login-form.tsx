import { useEffect, useMemo, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import axios from 'axios'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  login,
  mapLoginResponseFromRaw,
  type RawLoginResponse,
} from '@/features/auth/api/auth'
import { useAuthStore } from '@/features/auth/stores/auth-store'
import type { LoginResponse } from '@/features/auth/types'
import { env } from '@/libs/config/env'

interface GitHubOAuthMessage {
  source?: string
  payload?: (RawLoginResponse & { redirect_uri?: string }) | null
  error?: string
}

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
  const [isGitHubLoading, setIsGitHubLoading] = useState(false)
  const popupRef = useRef<Window | null>(null)
  const closeWatcherRef = useRef<number | null>(null)
  const desiredRedirectRef = useRef<string>('/prompts')
  const apiOrigin = useMemo(() => new URL(env.apiBaseUrl).origin, [])

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

  useEffect(() => {
    const handleMessage = (event: MessageEvent<GitHubOAuthMessage>) => {
      if (event.origin !== apiOrigin) {
        return
      }
      const data = event.data
      if (!data || data.source !== 'prompt-manager') {
        return
      }

      if (closeWatcherRef.current !== null) {
        window.clearInterval(closeWatcherRef.current)
        closeWatcherRef.current = null
      }

      if (popupRef.current && !popupRef.current.closed) {
        popupRef.current.close()
      }
      popupRef.current = null

      if (data.error) {
        setErrorMessage(data.error)
        setIsGitHubLoading(false)
        return
      }

      if (!data.payload) {
        setErrorMessage('GitHub 登录失败，请稍后重试')
        setIsGitHubLoading(false)
        return
      }

      try {
        const raw: RawLoginResponse = {
          tokens: data.payload.tokens,
          user: data.payload.user,
        }
        const mapped = mapLoginResponseFromRaw(raw)
        setAuth(mapped)

        const redirectFromPayload = data.payload.redirect_uri
        let target = desiredRedirectRef.current
        if (redirectFromPayload) {
          try {
            const parsed = new URL(redirectFromPayload)
            const combined = `${parsed.pathname}${parsed.search}${parsed.hash}`
            if (combined && combined !== '/') {
              target = combined
            }
          } catch (error) {
            console.warn('invalid redirect_uri from OAuth payload', error)
          }
        }

        setErrorMessage(null)
        navigate(target, { replace: true })
      } catch (error) {
        console.error('failed to process GitHub OAuth payload', error)
        setErrorMessage('GitHub 登录响应解析失败，请稍后重试')
      } finally {
        setIsGitHubLoading(false)
      }
    }

    window.addEventListener('message', handleMessage)
    return () => {
      window.removeEventListener('message', handleMessage)
    }
  }, [apiOrigin, navigate, setAuth])

  useEffect(() => {
    if (!isGitHubLoading || !popupRef.current) {
      return undefined
    }

    const timer = window.setInterval(() => {
      if (!popupRef.current) {
        return
      }
      if (popupRef.current.closed) {
        window.clearInterval(timer)
        closeWatcherRef.current = null
        popupRef.current = null
        setIsGitHubLoading(false)
        setErrorMessage((previous) => previous ?? 'GitHub 登录已取消')
      }
    }, 500)

    closeWatcherRef.current = timer

    return () => {
      window.clearInterval(timer)
      closeWatcherRef.current = null
    }
  }, [isGitHubLoading])

  const handleGitHubLogin = () => {
    setErrorMessage(null)
    const fallbackRedirect = (location.state as { from?: string } | null)?.from ?? '/prompts'
    desiredRedirectRef.current = fallbackRedirect

    const redirectURL = new URL(fallbackRedirect, window.location.origin).toString()
    const baseUrl = env.apiBaseUrl.replace(/\/$/, '')
    const authorizeURL = `${baseUrl}/auth/github/login?response_mode=web_message&redirect_uri=${encodeURIComponent(redirectURL)}`

    if (popupRef.current && !popupRef.current.closed) {
      popupRef.current.close()
    }
    popupRef.current = null

    if (closeWatcherRef.current !== null) {
      window.clearInterval(closeWatcherRef.current)
      closeWatcherRef.current = null
    }

    const width = 640
    const height = 720
    const left = window.screenX + Math.max(0, (window.outerWidth - width) / 2)
    const top = window.screenY + Math.max(0, (window.outerHeight - height) / 2)

    setIsGitHubLoading(true)

    const popup = window.open(
      authorizeURL,
      'prompt-manager-github-login',
      `width=${width},height=${height},left=${left},top=${top}`,
    )

    if (!popup) {
      setErrorMessage('浏览器阻止弹窗，请允许弹窗后重试')
      setIsGitHubLoading(false)
      return
    }

    popup.focus()
    popupRef.current = popup
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

      <div className="relative py-2 text-center text-xs text-slate-400">
        <span className="bg-white px-2">或</span>
      </div>

      <Button
        type="button"
        variant="outline"
        className="w-full"
        onClick={handleGitHubLogin}
        disabled={isSubmitting || isGitHubLoading}
      >
        {isGitHubLoading ? '正在跳转到 GitHub...' : '使用 GitHub 登录'}
      </Button>
    </form>
  )
}
