import { apiClient } from '@/libs/http/client'
import type {
  Prompt,
  PromptListMeta,
  PromptListParams,
  PromptListResult,
} from '@/features/prompts/types'

type RawPrompt = {
  id: string
  name: string
  description?: string | null
  tags?: unknown
  active_version_id?: string | null
  body?: string | null
  created_by?: string | null
  created_at: string
  updated_at: string
}

interface RawListMeta {
  total?: number
  limit?: number
  offset?: number
  hasMore?: boolean
}

interface RawListResponse {
  items: RawPrompt[]
  meta?: RawListMeta
}

interface SuccessResponse<T> {
  data: T
}

export async function listPrompts(
  params: PromptListParams = {},
): Promise<PromptListResult> {
  const response = await apiClient.get<SuccessResponse<RawListResponse>>(
    '/prompts',
    {
      params,
    },
  )

  const body = response.data.data
  const items = (body.items ?? []).map(mapPrompt)
  const meta = mapMeta(body.meta, params, items.length)

  return { items, meta }
}

export function mapPrompt(raw: RawPrompt): Prompt {
  return {
    id: raw.id,
    name: raw.name,
    description: raw.description ?? null,
    tags: parseTags(raw.tags),
    activeVersionId: raw.active_version_id ?? null,
    body: raw.body ?? null,
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

function mapMeta(
  raw: RawListMeta | undefined,
  params: PromptListParams,
  itemCount: number,
): PromptListMeta {
  const limit = raw?.limit ?? params.limit ?? 50
  const offset = raw?.offset ?? params.offset ?? 0
  const total = raw?.total ?? itemCount
  const hasMore = raw?.hasMore ?? offset+itemCount < total

  return {
    total,
    limit,
    offset,
    hasMore,
  }
}
