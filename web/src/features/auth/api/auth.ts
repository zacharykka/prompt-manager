import { apiClient } from '@/libs/http/client'
import type { LoginResponse, Tokens, User } from '@/features/auth/types'

interface RawTokens {
  access_token: string
  access_token_expires_at: string
  refresh_token: string
  refresh_token_expires_at: string
}

interface RawUser {
  id: string
  email: string
  role: string
  status: string
  last_login_at?: string | null
  created_at: string
  updated_at: string
}

interface RawLoginResponse {
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

export async function login(payload: LoginPayload): Promise<LoginResponse> {
  const response = await apiClient.post<SuccessResponse<RawLoginResponse>>(
    '/auth/login',
    payload,
  )
  const body = response.data.data
  return {
    tokens: mapTokens(body.tokens),
    user: mapUser(body.user),
  }
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
