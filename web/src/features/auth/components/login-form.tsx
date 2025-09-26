import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
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

const STORAGE_KEY = 'prompt-manager:oauth-result'
const POPUP_NAME = 'prompt-manager-github-login'
const HASH_PREFIX = '#pm_oauth='

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
  const githubHandledRef = useRef(false)
  const apiOrigin = useMemo(() => new URL(env.apiBaseUrl).origin, [])

  const cleanupPopup = useCallback(() => {
    if (closeWatcherRef.current !== null) {
      window.clearInterval(closeWatcherRef.current)
      closeWatcherRef.current = null
    }
    if (popupRef.current && !popupRef.current.closed) {
      popupRef.current.close()
    }
    popupRef.current = null
  }, [])

  const applyAuthResult = useCallback(
    (raw: RawLoginResponse, redirectFromPayload?: string | null) => {
      githubHandledRef.current = true
      cleanupPopup()
      setIsGitHubLoading(false)
      setErrorMessage(null)

      const mapped = mapLoginResponseFromRaw(raw)
      setAuth(mapped)

      const redirectOverride = redirectFromPayload ?? undefined
      let target = desiredRedirectRef.current
      if (redirectOverride) {
        try {
          const parsed = new URL(redirectOverride)
          const combined = `${parsed.pathname}${parsed.search}${parsed.hash}`
          if (combined && combined !== '/') {
            target = combined
          }
        } catch (error) {
          console.warn('invalid redirect_uri from OAuth payload', error)
        }
      }

      navigate(target, { replace: true })
    },
    [cleanupPopup, navigate, setAuth],
  )

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
      console.log('Received postMessage:', {
        origin: event.origin,
        expectedOrigin: apiOrigin,
        data: event.data,
        dataType: typeof event.data,
        hasSource: event.data?.source
      })

      // Temporarily allow messages from both origins for debugging
      const isValidOrigin = event.origin === apiOrigin ||
                           event.origin === window.location.origin ||
                           event.origin === 'http://localhost:8080' ||
                           event.origin === 'http://localhost:5173'

      if (!isValidOrigin) {
        console.warn('Message from unexpected origin:', event.origin, 'expected:', apiOrigin)
        return
      }

      const data = event.data
      // More specific check for our OAuth message format
      if (!data || typeof data !== 'object' || data.source !== 'prompt-manager') {
        console.log('Ignoring non-OAuth message:', data)
        return
      }

      console.log('Processing postMessage from OAuth popup')

      if (data.error) {
        githubHandledRef.current = true
        cleanupPopup()
        setIsGitHubLoading(false)
        setErrorMessage(data.error)
        return
      }

      if (!data.payload) {
        githubHandledRef.current = true
        cleanupPopup()
        setIsGitHubLoading(false)
        setErrorMessage('GitHub 登录失败，请稍后重试')
        return
      }

      try {
        const raw: RawLoginResponse = {
          tokens: data.payload.tokens,
          user: data.payload.user,
        }
        console.log('Applying auth result from postMessage')
        applyAuthResult(raw, data.payload.redirect_uri ?? null)
      } catch (error) {
        githubHandledRef.current = true
        cleanupPopup()
        setIsGitHubLoading(false)
        console.error('failed to process GitHub OAuth payload', error)
        setErrorMessage('GitHub 登录响应解析失败，请稍后重试')
      }
    }

    window.addEventListener('message', handleMessage)
    return () => {
      window.removeEventListener('message', handleMessage)
    }
  }, [apiOrigin, applyAuthResult, cleanupPopup])

  useEffect(() => {
    const handleStorage = (event: StorageEvent) => {
      if (event.key !== STORAGE_KEY || !event.newValue) {
        return
      }
      try {
        const decoded = JSON.parse(atob(event.newValue)) as RawLoginResponse & { redirect_uri?: string | null }
        applyAuthResult({ tokens: decoded.tokens, user: decoded.user }, decoded.redirect_uri ?? null)
      } catch (error) {
        console.error('failed to process OAuth storage payload', error)
        setErrorMessage('GitHub 登录响应解析失败，请稍后重试')
        githubHandledRef.current = true
        cleanupPopup()
        setIsGitHubLoading(false)
      } finally {
        window.localStorage.removeItem(STORAGE_KEY)
      }
    }

    window.addEventListener('storage', handleStorage)
    return () => {
      window.removeEventListener('storage', handleStorage)
    }
  }, [applyAuthResult, cleanupPopup])

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
        if (!githubHandledRef.current) {
          // Add a small delay to ensure localStorage write is complete
          setTimeout(() => {
            // Check for OAuth result in localStorage when popup closes
            const oauthResult = window.localStorage.getItem(STORAGE_KEY)
            console.log('Popup closed, checking localStorage (delayed):', { oauthResult: !!oauthResult })
            if (oauthResult) {
              try {
                const decoded = JSON.parse(atob(oauthResult)) as RawLoginResponse & { redirect_uri?: string | null }
                console.log('Processing OAuth result (delayed):', { user: decoded.user?.email, hasTokens: !!decoded.tokens })
                applyAuthResult({ tokens: decoded.tokens, user: decoded.user }, decoded.redirect_uri ?? null)
                window.localStorage.removeItem(STORAGE_KEY)
                return
              } catch (error) {
                console.error('failed to process OAuth storage payload', error)
                setErrorMessage('GitHub 登录响应解析失败，请稍后重试')
                window.localStorage.removeItem(STORAGE_KEY)
              }
            } else {
              // If still no result, try polling a few more times
              let retryCount = 0
              const retryTimer = window.setInterval(() => {
                const retryResult = window.localStorage.getItem(STORAGE_KEY)
                console.log(`Retry ${retryCount + 1}: checking localStorage:`, { oauthResult: !!retryResult })
                if (retryResult) {
                  window.clearInterval(retryTimer)
                  try {
                    const decoded = JSON.parse(atob(retryResult)) as RawLoginResponse & { redirect_uri?: string | null }
                    console.log('Processing OAuth result (retry):', { user: decoded.user?.email, hasTokens: !!decoded.tokens })
                    applyAuthResult({ tokens: decoded.tokens, user: decoded.user }, decoded.redirect_uri ?? null)
                    window.localStorage.removeItem(STORAGE_KEY)
                    return
                  } catch (error) {
                    console.error('failed to process OAuth storage payload (retry)', error)
                    setErrorMessage('GitHub 登录响应解析失败，请稍后重试')
                    window.localStorage.removeItem(STORAGE_KEY)
                  }
                }
                retryCount++
                if (retryCount >= 5) {
                  window.clearInterval(retryTimer)
                  console.log('OAuth retry attempts exhausted')
                  setIsGitHubLoading(false)
                }
              }, 200)
            }
            if (!oauthResult) {
              // Don't set loading to false immediately, let the retry logic handle it
              console.log('No OAuth result found, started retry polling')
            }
          }, 100)
        }
      }
    }, 300)

    closeWatcherRef.current = timer

    return () => {
      window.clearInterval(timer)
      closeWatcherRef.current = null
    }
  }, [isGitHubLoading, applyAuthResult])

  const handleGitHubLogin = () => {
    setErrorMessage(null)
    const fallbackRedirect = (location.state as { from?: string } | null)?.from ?? '/prompts'
    desiredRedirectRef.current = fallbackRedirect
    githubHandledRef.current = false

    const redirectURL = new URL(fallbackRedirect, window.location.origin).toString()
    const baseUrl = env.apiBaseUrl.replace(/\/$/, '')
    const authorizeURL = `${baseUrl}/auth/github/login?response_mode=web_message&redirect_uri=${encodeURIComponent(redirectURL)}&client_origin=${encodeURIComponent(window.location.origin)}`

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
      POPUP_NAME,
      `width=${width},height=${height},left=${left},top=${top},noopener=no`,
    )

    if (!popup) {
      setErrorMessage('浏览器阻止弹窗，请允许弹窗后重试')
      setIsGitHubLoading(false)
      return
    }

    popup.focus()
    popupRef.current = popup
  }

  useEffect(() => {
    const hash = window.location.hash
    console.log('Checking hash on mount:', hash)

    if (!hash.startsWith(HASH_PREFIX)) {
      return
    }

    console.log('Found OAuth hash, processing...')
    const encoded = decodeURIComponent(hash.slice(HASH_PREFIX.length))

    try {
      const decoded = JSON.parse(atob(encoded)) as RawLoginResponse & { redirect_uri?: string | null }
      console.log('Processing OAuth result from hash:', { user: decoded.user?.email, hasTokens: !!decoded.tokens })

      // Clear the hash first
      window.location.hash = ''

      // Always try localStorage first for hash-based auth (this is likely a popup redirect)
      console.log('Hash-based auth detected, using localStorage mechanism')
      try {
        window.localStorage.setItem(STORAGE_KEY, encoded)
        console.log('Stored OAuth data in localStorage')

        // If this seems to be a popup (URL came from backend redirect), close it
        if (window.history.length <= 2) {
          console.log('Appears to be popup redirect, closing window')
          setTimeout(() => {
            window.close()
          }, 100)
          return
        }
      } catch (error) {
        console.error('localStorage failed, processing directly', error)
      }

      // Fallback: process directly in current window
      console.log('Processing auth in current window')
      applyAuthResult({ tokens: decoded.tokens, user: decoded.user }, decoded.redirect_uri ?? null)
    } catch (error) {
      console.error('failed to process OAuth payload from hash', error)
      setErrorMessage('GitHub 登录响应解析失败，请稍后重试')
      window.location.hash = ''
    }
  }, [applyAuthResult])

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
