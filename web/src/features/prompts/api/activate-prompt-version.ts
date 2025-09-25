import { apiClient } from '@/libs/http/client'

export async function activatePromptVersion(promptId: string, versionId: string): Promise<void> {
  await apiClient.post(`/prompts/${promptId}/versions/${versionId}/activate`)
}
