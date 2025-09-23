import { apiClient } from '@/libs/http/client'
import type { Prompt } from '@/features/prompts/types'
import { mapPrompt } from '@/features/prompts/api/list-prompts'

type RawPromptResponse = {
  prompt: Parameters<typeof mapPrompt>[0]
}

type SuccessResponse<T> = {
  data: T
}

export async function getPrompt(promptId: string): Promise<Prompt> {
  const response = await apiClient.get<SuccessResponse<RawPromptResponse>>(
    `/prompts/${promptId}`,
  )
  return mapPrompt(response.data.data.prompt)
}
