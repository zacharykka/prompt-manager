import type { AxiosError } from 'axios'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { PromptTable } from '@/features/prompts/components/prompt-table'
import { usePromptsQuery } from '@/features/prompts/hooks/use-prompts'

const DEFAULT_LIMIT = 20

export function PromptListSection() {
  const {
    data,
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = usePromptsQuery({ limit: DEFAULT_LIMIT })

  if (isLoading) {
    return <PromptListSkeleton />
  }

  if (isError) {
    const axiosError = error as AxiosError<{ message?: string }>
    const message =
      axiosError.response?.data?.message ??
      axiosError.message ??
      'Prompt 列表加载失败，请稍后重试'
    return (
      <div className="space-y-4">
        <Alert variant="error">{message}</Alert>
        <Button variant="primary" onClick={() => refetch()} disabled={isFetching}>
          {isFetching ? '刷新中...' : '重新加载'}
        </Button>
      </div>
    )
  }

  const prompts = data ?? []

  if (prompts.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-slate-300 bg-white p-12 text-center">
        <h2 className="text-lg font-semibold text-slate-800">还没有 Prompt</h2>
        <p className="text-sm text-slate-500">
          开始创建第一个 Prompt 模板，便于团队共享与复用。
        </p>
        <div className="flex gap-2">
          <Button type="button">新建 Prompt</Button>
          <Button type="button" variant="secondary" onClick={() => refetch()}>
            刷新
          </Button>
        </div>
      </div>
    )
  }

  return <PromptTable prompts={prompts} />
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
