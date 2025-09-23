import { useQuery } from '@tanstack/react-query'

import { listPrompts } from '@/features/prompts/api/list-prompts'
import type { PromptListParams } from '@/features/prompts/types'

export function usePromptsQuery(params: PromptListParams = {}) {
  return useQuery({
    queryKey: ['prompts', params],
    queryFn: () => listPrompts(params),
    staleTime: params.includeDeleted ? 0 : 60_000,
    placeholderData: (previousData) => previousData,
  })
}
