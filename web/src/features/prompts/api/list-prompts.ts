import { apiClient } from '@/libs/http/client'
import type { Prompt, PromptListParams } from '@/features/prompts/types'

type RawPrompt = {
  id: string
  name: string
  description?: string | null
  tags?: unknown
  active_version_id?: string | null
  created_by?: string | null
  created_at: string
  updated_at: string
}

interface RawListResponse {
  items: RawPrompt[]
}

interface SuccessResponse<T> {
  data: T
}

export async function listPrompts(
  params: PromptListParams = {},
): Promise<Prompt[]> {
  const response = await apiClient.get<SuccessResponse<RawListResponse>>(
    '/prompts',
    {
      params,
    },
  )

  const body = response.data.data
  return (body.items ?? []).map(mapPrompt)
}

function mapPrompt(raw: RawPrompt): Prompt {
  return {
    id: raw.id,
    name: raw.name,
    description: raw.description ?? null,
    tags: parseTags(raw.tags),
    activeVersionId: raw.active_version_id ?? null,
    createdBy: raw.created_by ?? null,
    createdAt: raw.created_at,
    updatedAt: raw.updated_at,
  }
}

function parseTags(value: unknown): string[] {
  if (!value) {
    return []
  }
  if (Array.isArray(value)) {
    return value.filter((item): item is string => typeof item === 'string')
  }
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      if (Array.isArray(parsed)) {
        return parsed.filter((item): item is string => typeof item === 'string')
      }
    } catch {
      return []
    }
  }
  return []
}
