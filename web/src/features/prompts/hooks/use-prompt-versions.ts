import { useQuery } from '@tanstack/react-query'

import { getPromptVersionDiff, type PromptDiffParams } from '@/features/prompts/api/get-prompt-version-diff'
import { listPromptVersions } from '@/features/prompts/api/list-prompt-versions'
import type {
  PromptVersionDiff,
  PromptVersionListParams,
  PromptVersionListResult,
} from '@/features/prompts/types'

export function usePromptVersionsQuery(
  promptId: string | null,
  params: PromptVersionListParams = {},
) {
  return useQuery<PromptVersionListResult, Error>({
    queryKey: ['promptVersions', promptId, params],
    queryFn: () => {
      if (!promptId) {
        throw new Error('promptId is required')
      }
      return listPromptVersions(promptId, params)
    },
    enabled: Boolean(promptId),
    placeholderData: (previousData) => previousData,
  })
}

export function usePromptVersionDiff(
  promptId: string | null,
  versionId: string | null,
  params: PromptDiffParams = {},
) {
  return useQuery<PromptVersionDiff, Error>({
    queryKey: ['promptVersionDiff', promptId, versionId, params],
    queryFn: () => {
      if (!promptId || !versionId) {
        throw new Error('promptId and versionId are required')
      }
      return getPromptVersionDiff(promptId, versionId, params)
    },
    enabled: Boolean(promptId && versionId),
  })
}
