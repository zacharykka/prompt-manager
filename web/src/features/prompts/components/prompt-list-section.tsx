import { useState, type FormEvent } from 'react'
import type { AxiosError } from 'axios'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PromptTable } from '@/features/prompts/components/prompt-table'
import { usePromptsQuery } from '@/features/prompts/hooks/use-prompts'

const DEFAULT_LIMIT = 20

interface PromptListSectionProps {
  onCreatePrompt?: () => void
}

export function PromptListSection({ onCreatePrompt }: PromptListSectionProps) {
  const [searchInput, setSearchInput] = useState('')
  const [queryState, setQueryState] = useState({ search: '', offset: 0 })

  const {
    data,
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = usePromptsQuery({
    limit: DEFAULT_LIMIT,
    offset: queryState.offset,
    search: queryState.search,
  })

  const isInitialLoading = isLoading && !data
  const items = data?.items ?? []
  const meta = data?.meta ?? {
    total: items.length,
    limit: DEFAULT_LIMIT,
    offset: queryState.offset,
    hasMore: false,
  }

  const handleSearchSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setQueryState({ search: searchInput.trim(), offset: 0 })
  }

  const handleReset = () => {
    setSearchInput('')
    setQueryState({ search: '', offset: 0 })
  }

  const handlePrevPage = () => {
    setQueryState((prev) => ({
      search: prev.search,
      offset: Math.max(0, prev.offset - DEFAULT_LIMIT),
    }))
  }

  const handleNextPage = () => {
    setQueryState((prev) => ({
      search: prev.search,
      offset: prev.offset + DEFAULT_LIMIT,
    }))
  }

  const hasPrev = (meta.offset ?? 0) > 0
  const hasNext = meta.hasMore
  const totalPages = Math.max(1, Math.ceil((meta.total ?? 0) / (meta.limit || DEFAULT_LIMIT)))
  const currentPage = Math.min(
    totalPages,
    Math.floor((meta.offset ?? 0) / (meta.limit || DEFAULT_LIMIT)) + 1,
  )

  return (
    <div className="space-y-5">
      <form
        onSubmit={handleSearchSubmit}
        className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm md:flex-row md:items-center md:justify-between"
      >
        <div className="flex w-full flex-col gap-2 md:flex-row md:items-center md:gap-3">
          <Input
            type="search"
            placeholder="按名称搜索 Prompt"
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
            className="md:w-72"
          />
          <div className="flex gap-2">
            <Button type="submit" disabled={isFetching}>
              搜索
            </Button>
            <Button
              type="button"
              variant="secondary"
              onClick={handleReset}
              disabled={isFetching && queryState.search === '' && queryState.offset === 0}
            >
              重置
            </Button>
          </div>
        </div>
        <div className="text-sm text-slate-500">
          共 {meta.total ?? 0} 条记录
          {queryState.search ? `，关键词 “${queryState.search}”` : ''}
        </div>
      </form>

      {isInitialLoading ? (
        <PromptListSkeleton />
      ) : isError ? (
        <PromptListError error={error} onRetry={() => refetch()} isFetching={isFetching} />
      ) : items.length === 0 ? (
        <PromptListEmpty onCreatePrompt={onCreatePrompt} onRefresh={() => refetch()} />
      ) : (
        <div className="space-y-4">
          <PromptTable prompts={items} />
          <div className="flex flex-col gap-3 border-t border-slate-200 pt-4 md:flex-row md:items-center md:justify-between">
            <span className="text-sm text-slate-500">
              第 {currentPage} / {totalPages} 页
            </span>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="secondary"
                onClick={handlePrevPage}
                disabled={!hasPrev || isFetching}
              >
                上一页
              </Button>
              <Button
                type="button"
                variant="secondary"
                onClick={handleNextPage}
                disabled={!hasNext || isFetching}
              >
                下一页
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function PromptListError({
  error,
  onRetry,
  isFetching,
}: {
  error: unknown
  onRetry: () => void
  isFetching: boolean
}) {
  const axiosError = error as AxiosError<{ message?: string }>
  const message =
    axiosError?.response?.data?.message ??
    axiosError?.message ??
    'Prompt 列表加载失败，请稍后重试'

  return (
    <div className="space-y-4">
      <Alert variant="error">{message}</Alert>
      <Button variant="primary" onClick={onRetry} disabled={isFetching}>
        {isFetching ? '刷新中...' : '重新加载'}
      </Button>
    </div>
  )
}

function PromptListEmpty({
  onCreatePrompt,
  onRefresh,
}: {
  onCreatePrompt?: () => void
  onRefresh: () => void
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-slate-300 bg-white p-12 text-center">
      <h2 className="text-lg font-semibold text-slate-800">还没有 Prompt</h2>
      <p className="text-sm text-slate-500">
        开始创建第一个 Prompt 模板，便于团队共享与复用。
      </p>
      <div className="flex gap-2">
        {onCreatePrompt ? (
          <Button type="button" onClick={onCreatePrompt}>
            新建 Prompt
          </Button>
        ) : null}
        <Button type="button" variant="secondary" onClick={onRefresh}>
          刷新
        </Button>
      </div>
    </div>
  )
}

function PromptListSkeleton() {
  return (
    <div className="space-y-3">
      <div className="h-6 w-52 animate-pulse rounded bg-slate-200" />
      <div className="space-y-2 rounded-xl border border-slate-200 bg-white p-4">
        {Array.from({ length: 5 }).map((_, index) => (
          <div key={index} className="flex items-center gap-4">
            <div className="h-6 w-1/4 animate-pulse rounded bg-slate-200" />
            <div className="h-6 w-1/3 animate-pulse rounded bg-slate-200" />
            <div className="h-6 w-20 animate-pulse rounded bg-slate-200" />
            <div className="h-6 w-24 animate-pulse rounded bg-slate-200" />
            <div className="h-6 w-32 animate-pulse rounded bg-slate-200" />
          </div>
        ))}
      </div>
    </div>
  )
}
