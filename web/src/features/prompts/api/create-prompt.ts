import { apiClient } from '@/libs/http/client'
import { mapPrompt } from '@/features/prompts/api/list-prompts'
import type { Prompt } from '@/features/prompts/types'

interface CreatePromptPayload {
  name: string
  description?: string
  tags?: string[]
  body?: string
}

interface RawPromptResponse {
  prompt: {
    id: string
    name: string
    description?: string | null
    tags?: unknown
    active_version_id?: string | null
    created_by?: string | null
    created_at: string
    updated_at: string
  }
}

interface SuccessResponse<T> {
  data: T
}

export async function createPrompt(payload: CreatePromptPayload): Promise<Prompt> {
  const response = await apiClient.post<SuccessResponse<RawPromptResponse>>(
    '/prompts',
    payload,
  )

  return mapPrompt(response.data.data.prompt)
}
