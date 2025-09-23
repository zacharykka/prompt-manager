import { apiClient } from '@/libs/http/client'

export interface CreatePromptVersionPayload {
  body: string
  variables_schema?: unknown
  metadata?: unknown
  status?: 'draft' | 'published' | 'archived'
  activate?: boolean
}

export async function createPromptVersion(
  promptId: string,
  payload: CreatePromptVersionPayload,
): Promise<void> {
  await apiClient.post(`/prompts/${promptId}/versions`, payload)
}
