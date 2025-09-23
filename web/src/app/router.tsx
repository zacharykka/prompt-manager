import { Suspense } from 'react'
import {
  RouterProvider,
  Navigate,
  createBrowserRouter,
} from 'react-router-dom'
import type { RouteObject } from 'react-router-dom'

import { AuthLayout } from './layouts/auth-layout'
import { RootLayout } from './layouts/root-layout'
import { RequireAuth } from './require-auth'
import { LoadingScreen } from '@/components/loading-screen'
import { LoginPage } from '@/features/auth/pages/login-page'
import { DashboardPage } from '@/features/dashboard/pages/dashboard-page'

const routes: RouteObject[] = [
  {
    path: '/',
    element: (
      <RequireAuth>
        <RootLayout />
      </RequireAuth>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="/prompts" replace />,
      },
      {
        path: 'prompts',
        element: <DashboardPage />,
      },
    ],
  },
  {
    path: '/auth',
    element: <AuthLayout />,
    children: [
      {
        path: 'login',
        element: <LoginPage />,
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/prompts" replace />,
  },
]

const router = createBrowserRouter(routes)

export function AppRouter() {
  return (
    <Suspense fallback={<LoadingScreen />}>
      <RouterProvider router={router} />
    </Suspense>
  )
}
