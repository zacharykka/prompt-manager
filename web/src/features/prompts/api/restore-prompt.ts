import { apiClient } from '@/libs/http/client'

export async function restorePrompt(promptId: string): Promise<void> {
  await apiClient.post(`/prompts/${promptId}/restore`)
}
