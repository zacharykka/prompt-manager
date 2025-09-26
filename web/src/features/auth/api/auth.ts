import axios from 'axios'

import type { LoginResponse, Tokens, User } from '@/features/auth/types'
import { env } from '@/libs/config/env'

export interface RawTokens {
  access_token: string
  access_token_expires_at: string
  refresh_token: string
  refresh_token_expires_at: string
}

export interface RawUser {
  id: string
  email: string
  role: string
  status: string
  last_login_at?: string | null
  created_at: string
  updated_at: string
}

export interface RawLoginResponse {
  tokens: RawTokens
  user: RawUser
}

export interface LoginPayload {
  email: string
  password: string
}

interface SuccessResponse<T> {
  data: T
}

const authClient = axios.create({
  baseURL: env.apiBaseUrl,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

export async function login(payload: LoginPayload): Promise<LoginResponse> {
  const response = await authClient.post<SuccessResponse<RawLoginResponse>>(
    '/auth/login',
    payload,
  )
  return mapLoginResponseFromRaw(response.data.data)
}

export async function refreshTokens(refreshToken: string): Promise<LoginResponse> {
  const response = await authClient.post<SuccessResponse<RawLoginResponse>>(
    '/auth/refresh',
    { refresh_token: refreshToken },
  )
  return mapLoginResponseFromRaw(response.data.data)
}

function mapTokens(raw: RawTokens): Tokens {
  return {
    accessToken: raw.access_token,
    accessTokenExpiresAt: raw.access_token_expires_at,
    refreshToken: raw.refresh_token,
    refreshTokenExpiresAt: raw.refresh_token_expires_at,
  }
}

function mapUser(raw: RawUser): User {
  return {
    id: raw.id,
    email: raw.email,
    role: raw.role as User['role'],
    status: raw.status as User['status'],
    lastLoginAt: raw.last_login_at ?? null,
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
  }
}

export function mapLoginResponseFromRaw(raw: RawLoginResponse): LoginResponse {
  return {
    tokens: mapTokens(raw.tokens),
    user: mapUser(raw.user),
  }
}
