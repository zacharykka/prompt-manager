import { apiClient } from '@/libs/http/client'

export async function deletePrompt(promptId: string): Promise<void> {
  await apiClient.delete(`/prompts/${promptId}`)
}
