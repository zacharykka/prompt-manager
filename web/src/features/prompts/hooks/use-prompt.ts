import { useQuery } from '@tanstack/react-query'

import { getPrompt } from '@/features/prompts/api/get-prompt'

export function usePromptQuery(promptId: string | undefined) {
  return useQuery({
    queryKey: ['prompt', promptId],
    queryFn: () => {
      if (!promptId) {
        throw new Error('promptId is required')
      }
      return getPrompt(promptId)
    },
    enabled: Boolean(promptId),
    staleTime: 30_000,
  })
}
