export type PromptStatus = 'active' | 'deleted'

export interface Prompt {
  id: string
  name: string
  description: string | null
  tags: string[]
  activeVersionId: string | null
  body: string | null
  createdBy: string | null
  createdAt: string
  updatedAt: string
  status: PromptStatus
  deletedAt: string | null
}

export interface PromptListParams {
  limit?: number
  offset?: number
  search?: string
  includeDeleted?: boolean
}

export interface PromptListMeta {
  total: number
  limit: number
  offset: number
  hasMore: boolean
}

export interface PromptListResult {
  items: Prompt[]
  meta: PromptListMeta
}

export interface PromptVersion {
  id: string
  versionNumber: number
  body: string
  variablesSchema: Record<string, unknown> | null
  metadata: Record<string, unknown> | null
  status: string
  createdBy: string | null
  createdAt: string
}

export interface PromptVersionSummary {
  id: string
  versionNumber: number
  status: string
  createdBy: string | null
  createdAt: string
}

export interface PromptVersionDiffSegment {
  type: 'equal' | 'insert' | 'delete'
  text: string
}

export type PromptVersionDiffChangeType = 'added' | 'removed' | 'modified'

export interface PromptVersionDiffChange {
  key: string
  type: PromptVersionDiffChangeType
  left?: string
  right?: string
}

export interface PromptVersionDiffField {
  changes: PromptVersionDiffChange[]
}

export interface PromptVersionDiff {
  promptId: string
  base: PromptVersionSummary
  target: PromptVersionSummary
  body: PromptVersionDiffSegment[]
  variablesSchema?: PromptVersionDiffField
  metadata?: PromptVersionDiffField
}

export type PromptVersionStatus = 'draft' | 'published' | 'archived'

export interface PromptVersionListParams {
  limit?: number
  offset?: number
  status?: PromptVersionStatus
}

export interface PromptVersionListMeta {
  limit: number
  offset: number
  hasMore: boolean
}

export interface PromptVersionListResult {
  items: PromptVersion[]
  meta?: PromptVersionListMeta
}
