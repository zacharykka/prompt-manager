import { apiClient } from '@/libs/http/client'
import type {
  PromptVersion,
  PromptVersionListMeta,
  PromptVersionListParams,
  PromptVersionListResult,
} from '@/features/prompts/types'

type RawPromptVersion = {
  id: string
  version_number: number
  body: string
  variables_schema?: unknown
  metadata?: unknown
  status: string
  created_by?: string | null
  created_at: string
}

interface RawListResponse {
  items: RawPromptVersion[]
  meta?: {
    limit: number
    offset: number
    has_more: boolean
  }
}

interface SuccessResponse<T> {
  data: T
}

function toRecord(value: unknown): Record<string, unknown> | null {
  if (value == null) {
    return null
  }
  if (typeof value === 'object' && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>
      }
    } catch {
      return null
    }
  }
  return null
}

function mapPromptVersion(raw: RawPromptVersion): PromptVersion {
  return {
    id: raw.id,
    versionNumber: raw.version_number,
    body: raw.body,
    variablesSchema: toRecord(raw.variables_schema),
    metadata: toRecord(raw.metadata),
    status: raw.status,
    createdBy: raw.created_by ?? null,
    createdAt: raw.created_at,
  }
}

export async function listPromptVersions(
  promptId: string,
  params: PromptVersionListParams = {},
): Promise<PromptVersionListResult> {
  const response = await apiClient.get<SuccessResponse<RawListResponse>>(
    `/prompts/${promptId}/versions`,
    {
      params: {
        limit: params.limit,
        offset: params.offset,
        status: params.status,
      },
    },
  )

  const raw = response.data.data
  const result: PromptVersionListResult = {
    items: (raw.items ?? []).map(mapPromptVersion),
  }
  if (raw.meta) {
    const meta: PromptVersionListMeta = {
      limit: raw.meta.limit,
      offset: raw.meta.offset,
      hasMore: raw.meta.has_more,
    }
    result.meta = meta
  }
  return result
}
