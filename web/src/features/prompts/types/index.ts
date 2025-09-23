export interface Prompt {
  id: string
  name: string
  description: string | null
  tags: string[]
  activeVersionId: string | null
  createdBy: string | null
  createdAt: string
  updatedAt: string
}

export interface PromptListParams {
  limit?: number
  offset?: number
}
