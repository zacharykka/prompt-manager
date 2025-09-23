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
