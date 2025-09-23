import { apiClient } from '@/libs/http/client'

export interface UpdatePromptPayload {
  name?: string
  description?: string | null
  tags?: string[]
}

export async function updatePrompt(
  promptId: string,
  payload: UpdatePromptPayload,
): Promise<void> {
  await apiClient.patch(`/prompts/${promptId}`, payload)
}
