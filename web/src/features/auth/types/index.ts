export interface User {
  id: string
  email: string
  role: 'admin' | 'editor' | 'viewer'
  status: 'active' | 'disabled'
  lastLoginAt?: string | null
  createdAt: string
  updatedAt: string
}

export interface Tokens {
  accessToken: string
  accessTokenExpiresAt: string
  refreshToken: string
  refreshTokenExpiresAt: string
}

export interface LoginResponse {
  tokens: Tokens
  user: User
}
