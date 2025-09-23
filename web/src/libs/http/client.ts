import type { AxiosError, AxiosRequestConfig } from 'axios'
import axios from 'axios'

import { refreshTokens } from '@/features/auth/api/auth'
import { useAuthStore } from '@/features/auth/stores/auth-store'
import type { LoginResponse } from '@/features/auth/types'
import { env } from '@/libs/config/env'

export const apiClient = axios.create({
  baseURL: env.apiBaseUrl,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

type RetryableRequestConfig = AxiosRequestConfig & { _retry?: boolean }

let refreshPromise: Promise<LoginResponse> | null = null

apiClient.interceptors.request.use((config) => {
  const accessToken = useAuthStore.getState().tokens?.accessToken
  if (accessToken) {
    config.headers = config.headers ?? {}
    config.headers.Authorization = `Bearer ${accessToken}`
  }
  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const status = error.response?.status
    const originalRequest = error.config as RetryableRequestConfig | undefined

    if (status === 401 && originalRequest && !originalRequest._retry) {
      const initialState = useAuthStore.getState()
      const refreshToken = initialState.tokens?.refreshToken

      if (!refreshToken) {
        initialState.clearAuth()
        return Promise.reject(error)
      }

      if (originalRequest.url?.includes('/auth/refresh')) {
        initialState.clearAuth()
        return Promise.reject(error)
      }

      try {
        if (!refreshPromise) {
          refreshPromise = refreshTokens(refreshToken)
            .then((result) => {
              const { setAuth, user: existingUser } = useAuthStore.getState()

              const nextUser = result.user ?? existingUser
              if (!nextUser) {
                throw new Error('missing user information after refresh')
              }

              setAuth({ user: nextUser, tokens: result.tokens })

              return { user: nextUser, tokens: result.tokens }
            })
            .catch((refreshError) => {
              useAuthStore.getState().clearAuth()
              throw refreshError
            })
            .finally(() => {
              refreshPromise = null
            })
        }

        const refreshed = await refreshPromise
        originalRequest._retry = true
        originalRequest.headers = {
          ...(originalRequest.headers ?? {}),
          Authorization: `Bearer ${refreshed.tokens.accessToken}`,
        }
        return apiClient(originalRequest)
      } catch (refreshError) {
        return Promise.reject(refreshError)
      }
    }

    if (status === 401) {
      useAuthStore.getState().clearAuth()
    }

    return Promise.reject(error)
  },
)
