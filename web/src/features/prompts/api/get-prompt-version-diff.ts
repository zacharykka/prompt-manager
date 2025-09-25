import { apiClient } from '@/libs/http/client'
import type {
  PromptVersionDiff,
  PromptVersionDiffChange,
  PromptVersionDiffField,
  PromptVersionDiffSegment,
  PromptVersionSummary,
} from '@/features/prompts/types'

interface RawDiffChange {
  key: string
  type: 'added' | 'removed' | 'modified'
  left?: string
  right?: string
}

interface RawDiffField {
  changes: RawDiffChange[]
}

interface RawDiffSummary {
  id: string
  version_number: number
  status: string
  created_by?: string | null
  created_at: string
}

interface RawDiffSegment {
  type: 'equal' | 'insert' | 'delete'
  text: string
}

interface RawDiff {
  prompt_id: string
  base: RawDiffSummary
  target: RawDiffSummary
  body: RawDiffSegment[]
  variables_schema?: RawDiffField
  metadata?: RawDiffField
}

interface SuccessResponse<T> {
  data: {
    diff: T
  }
}

export interface PromptDiffParams {
  targetVersionId?: string
  compareTo?: 'active' | 'previous'
}

function mapSummary(raw: RawDiffSummary): PromptVersionSummary {
  return {
    id: raw.id,
    versionNumber: raw.version_number,
    status: raw.status,
    createdBy: raw.created_by ?? null,
    createdAt: raw.created_at,
  }
}

function mapField(raw?: RawDiffField): PromptVersionDiffField | undefined {
  if (!raw) {
    return undefined
  }
  const changes: PromptVersionDiffChange[] = (raw.changes ?? []).map((change) => ({
    key: change.key,
    type: change.type,
    left: change.left,
    right: change.right,
  }))
  if (changes.length === 0) {
    return undefined
  }
  return { changes }
}

function mapSegments(raw: RawDiffSegment[]): PromptVersionDiffSegment[] {
  return raw.map((segment) => ({
    type: segment.type,
    text: segment.text,
  }))
}

export async function getPromptVersionDiff(
  promptId: string,
  versionId: string,
  params: PromptDiffParams = {},
): Promise<PromptVersionDiff> {
  const response = await apiClient.get<SuccessResponse<RawDiff>>(
    `/prompts/${promptId}/versions/${versionId}/diff`,
    {
      params: params.targetVersionId
        ? { targetVersionId: params.targetVersionId }
        : params.compareTo === 'active'
          ? { compareTo: 'active' }
          : {},
    },
  )

  const raw = response.data.data.diff
  return {
    promptId: raw.prompt_id,
    base: mapSummary(raw.base),
    target: mapSummary(raw.target),
    body: mapSegments(raw.body ?? []),
    variablesSchema: mapField(raw.variables_schema),
    metadata: mapField(raw.metadata),
  }
}
