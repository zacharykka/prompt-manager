import { useEffect, useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { AxiosError } from 'axios'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PromptTable } from '@/features/prompts/components/prompt-table'
import { DeletePromptDialog } from '@/features/prompts/components/delete-prompt-dialog'
import { deletePrompt } from '@/features/prompts/api/delete-prompt'
import { usePromptsQuery } from '@/features/prompts/hooks/use-prompts'
import type { Prompt } from '@/features/prompts/types'

const DEFAULT_LIMIT = 20
type PromptView = 'active' | 'deleted'

interface PromptListSectionProps {
  onCreatePrompt?: () => void
  onEditPrompt?: (prompt: Prompt) => void
}

export function PromptListSection({ onCreatePrompt, onEditPrompt }: PromptListSectionProps) {
  const location = useLocation()
  const navigate = useNavigate()
  const [searchInput, setSearchInput] = useState('')
  const [queryState, setQueryState] = useState({ search: '', offset: 0 })
  const [view, setView] = useState<PromptView>('active')
  const [feedback, setFeedback] = useState<
    | {
        type: 'success' | 'error'
        message: string
      }
    | null
  >(null)
  const [deleteTarget, setDeleteTarget] = useState<Prompt | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const queryClient = useQueryClient()

  useEffect(() => {
    const state = location.state as { feedback?: { type: 'success' | 'error'; message: string } } | undefined
    if (state?.feedback) {
      setFeedback(state.feedback)
      navigate(location.pathname + location.search, { replace: true })
    }
  }, [location, navigate])

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
    includeDeleted: view === 'deleted',
  })

  const isInitialLoading = isLoading && !data
  const items = data?.items ?? []
  const filteredItems = view === 'deleted'
    ? items.filter((item) => item.status === 'deleted')
    : items.filter((item) => item.status !== 'deleted')
  const meta = view === 'deleted'
    ? {
        total: filteredItems.length,
        limit: filteredItems.length,
        offset: 0,
        hasMore: false,
      }
    : data?.meta ?? {
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

  const handleViewChange = (nextView: PromptView) => {
    if (view === nextView) {
      return
    }
    setView(nextView)
    setQueryState((prev) => ({ search: prev.search, offset: 0 }))
  }

  const {
    mutateAsync: deletePromptAsync,
    isPending: isDeleting,
  } = useMutation({
    mutationFn: (promptId: string) => deletePrompt(promptId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['prompts'] })
    },
  })

  const handleDeletePrompt = (prompt: Prompt) => {
    setDeleteError(null)
    setDeleteTarget(prompt)
  }

  const handleConfirmDelete = async () => {
    if (!deleteTarget) {
      return
    }
    setDeleteError(null)
    try {
      await deletePromptAsync(deleteTarget.id)
      setFeedback({ type: 'success', message: `Prompt “${deleteTarget.name}” 已删除。` })
      setDeleteTarget(null)
    } catch (error) {
      const message = parseDeleteError(error)
      setDeleteError(message)
      setFeedback({ type: 'error', message })
    }
  }

  const handleCloseDialog = () => {
    if (isDeleting) {
      return
    }
    setDeleteTarget(null)
    setDeleteError(null)
  }

  return (
    <div className="space-y-5">
      {feedback ? (
        <Alert
          variant={feedback.type === 'success' ? 'success' : 'error'}
          className="flex items-center justify-between gap-4"
        >
          <span>{feedback.message}</span>
          <Button variant="ghost" size="sm" onClick={() => setFeedback(null)}>
            知道了
          </Button>
        </Alert>
      ) : null}

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
        <div className="flex-shrink-0 text-sm font-medium text-slate-500 whitespace-nowrap md:text-base">
          共 {meta.total ?? 0} 条记录{queryState.search ? `，关键词 “${queryState.search}”` : ''}
        </div>
      </form>

      <div className="flex items-center justify-between rounded-xl border border-slate-200 bg-white p-3">
        <span className="text-sm font-medium text-slate-600">视图</span>
        <div className="flex gap-2">
          <Button
            type="button"
            size="sm"
            variant={view === 'active' ? 'primary' : 'secondary'}
            onClick={() => handleViewChange('active')}
            disabled={isFetching && view === 'active'}
          >
            全部
          </Button>
          <Button
            type="button"
            size="sm"
            variant={view === 'deleted' ? 'primary' : 'secondary'}
            onClick={() => handleViewChange('deleted')}
            disabled={isFetching && view === 'deleted'}
          >
            回收站
          </Button>
        </div>
      </div>

      {isInitialLoading ? (
        <PromptListSkeleton />
      ) : isError ? (
        <PromptListError error={error} onRetry={() => refetch()} isFetching={isFetching} />
      ) : filteredItems.length === 0 ? (
        <PromptListEmpty
          view={view}
          onCreatePrompt={onCreatePrompt}
          onRefresh={() => refetch()}
        />
      ) : (
        <div className="space-y-4">
          <PromptTable
            prompts={filteredItems}
            onDeletePrompt={view === 'deleted' ? undefined : handleDeletePrompt}
            deletingPromptId={isDeleting ? deleteTarget?.id ?? null : null}
            disableActions={isDeleting || view === 'deleted'}
            onEditPrompt={view === 'deleted' ? undefined : onEditPrompt}
          />
          {view === 'deleted' ? null : (
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
          )}
        </div>
      )}

      <DeletePromptDialog
        open={Boolean(deleteTarget)}
        prompt={deleteTarget}
        errorMessage={deleteError}
        isProcessing={isDeleting}
        onCancel={handleCloseDialog}
        onConfirm={handleConfirmDelete}
      />
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

function parseDeleteError(error: unknown): string {
  const axiosError = error as AxiosError<{ message?: string }> | undefined
  const status = axiosError?.response?.status
  if (status === 404) {
    return '目标 Prompt 已删除或不存在。'
  }
  if (status === 403) {
    return '您没有权限删除该 Prompt，请联系管理员。'
  }
  return (
    axiosError?.response?.data?.message ??
    axiosError?.message ??
    '删除 Prompt 失败，请稍后重试。'
  )
}

function PromptListEmpty({
  view,
  onCreatePrompt,
  onRefresh,
}: {
  view: PromptView
  onCreatePrompt?: () => void
  onRefresh: () => void
}) {
  const title = view === 'deleted' ? '回收站暂无内容' : '还没有 Prompt'
  const description =
    view === 'deleted'
      ? '已删除的 Prompt 会出现在这里，暂时没有可展示的记录。'
      : '开始创建第一个 Prompt 模板，便于团队共享与复用。'
  return (
    <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-slate-300 bg-white p-12 text-center">
      <h2 className="text-lg font-semibold text-slate-800">{title}</h2>
      <p className="text-sm text-slate-500">{description}</p>
      <div className="flex gap-2">
        {view === 'deleted' ? null : onCreatePrompt ? (
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
