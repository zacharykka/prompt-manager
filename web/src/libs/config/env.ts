const DEFAULT_API_BASE_URL = 'http://localhost:8080/api/v1'

function getEnvVar(key: string, fallback: string): string {
  const value = import.meta.env[key as keyof ImportMetaEnv]
  return typeof value === 'string' && value.length > 0 ? value : fallback
}

export const env = {
  apiBaseUrl: getEnvVar('VITE_API_BASE_URL', DEFAULT_API_BASE_URL),
}
