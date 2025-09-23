import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

import type { Tokens, User } from '../types'

interface AuthState {
  user: User | null
  tokens: Tokens | null
  isAuthenticated: () => boolean
  setAuth: (payload: { user: User; tokens: Tokens }) => void
  updateTokens: (tokens: Tokens) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      tokens: null,
      isAuthenticated: () => Boolean(get().tokens?.accessToken),
      setAuth: ({ user, tokens }) => set({ user, tokens }),
      updateTokens: (tokens) => set({ tokens }),
      clearAuth: () => set({ user: null, tokens: null }),
    }),
    {
      name: 'prompt-manager-auth',
      storage: createJSONStorage(() => localStorage),
      partialize: ({ user, tokens }) => ({ user, tokens }),
    },
  ),
)
