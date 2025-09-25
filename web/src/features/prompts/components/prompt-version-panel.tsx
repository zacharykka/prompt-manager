import { useEffect, useMemo, useState } from 'react'
import { format } from 'date-fns'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { AxiosError } from 'axios'

import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { activatePromptVersion } from '@/features/prompts/api/activate-prompt-version'
import { usePromptVersionDiff, usePromptVersionsQuery } from '@/features/prompts/hooks/use-prompt-versions'

interface PromptVersionPanelProps {
  promptId: string
  activeVersionId: string | null
  promptName?: string
}

type CompareMode = 'previous' | 'active'

type FeedbackState = {
  type: 'success' | 'error'
  message: string
} | null

export function PromptVersionPanel({ promptId, activeVersionId, promptName }: PromptVersionPanelProps) {
  const queryClient = useQueryClient()
  const [selectedVersionId, setSelectedVersionId] = useState<string | null>(null)
  const [compareMode, setCompareMode] = useState<CompareMode>('previous')
  const [feedback, setFeedback] = useState<FeedbackState>(null)

  const { data, isLoading, isError, error, refetch } = usePromptVersionsQuery(promptId)
  const versionItems = data?.items
  const versions = useMemo(() => versionItems ?? [], [versionItems])
  const defaultVersionId = useMemo(() => (versions.length > 0 ? versions[0].id : null), [versions])

  useEffect(() => {
    if (!selectedVersionId && defaultVersionId) {
      setSelectedVersionId(defaultVersionId)
    }
  }, [selectedVersionId, defaultVersionId])

  const diffParams = useMemo(() => {
    if (compareMode === 'active') {
      return { compareTo: 'active' as const }
    }
    return {}
  }, [compareMode])

  const {
    data: diff,
    isLoading: isDiffLoading,
    isError: isDiffError,
    error: diffError,
  } = usePromptVersionDiff(promptId, selectedVersionId, diffParams)

  const activateMutation = useMutation({
    mutationFn: (versionId: string) => activatePromptVersion(promptId, versionId),
    onSuccess: async () => {
      setFeedback({ type: 'success', message: `版本已设为当前版本${promptName ? `：${promptName}` : ''}。` })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['prompt', promptId] }),
        queryClient.invalidateQueries({ queryKey: ['promptVersions', promptId] }),
        queryClient.invalidateQueries({ queryKey: ['prompts'] }),
      ])
    },
    onError: (err: unknown) => {
      setFeedback({ type: 'error', message: parseActivationError(err) })
    },
  })

  const handleActivate = async (versionId: string) => {
    await activateMutation.mutateAsync(versionId)
  }

  const handleSelectVersion = (versionId: string) => {
    setSelectedVersionId(versionId)
  }

  const renderDiffBody = () => {
    if (!selectedVersionId) {
      return <p className="text-sm text-slate-500">请选择一个版本查看差异。</p>
    }
    if (isDiffLoading) {
      return <p className="text-sm text-slate-500">差异计算中...</p>
    }
    if (isDiffError) {
      const axiosError = diffError as AxiosError<{ message?: string }> | undefined
      const status = axiosError?.response?.status
      if (status === 404 && compareMode === 'previous') {
        return <p className="text-sm text-slate-500">没有更早的版本可供比较。</p>
      }
      const message =
        axiosError?.response?.data?.message ?? axiosError?.message ?? '无法获取差异结果，请稍后再试。'
      return (
        <Alert variant="error" className="text-sm">
          {message}
        </Alert>
      )
    }
    if (!diff) {
      return <p className="text-sm text-slate-500">暂无差异信息。</p>
    }

    const hasBodyChanges = diff.body.some((segment) => segment.type !== 'equal')
    if (!hasBodyChanges) {
      return <p className="text-sm text-slate-500">正文与比较版本一致。</p>
    }

    return (
      <div className="rounded-xl bg-slate-950 p-4 font-mono text-xs text-slate-100">
        {diff.body.map((segment, index) => {
          const className =
            segment.type === 'insert'
              ? 'text-emerald-300'
              : segment.type === 'delete'
                ? 'text-rose-300 line-through'
                : 'text-slate-300'
          return (
            <span key={`${segment.type}-${index}`} className={className}>
              {segment.text}
            </span>
          )
        })}
      </div>
    )
  }

  const renderFieldDiff = (title: string, changesField?: { changes: Array<{ key: string; type: string; left?: string; right?: string }> }) => {
    if (!changesField || changesField.changes.length === 0) {
      return null
    }

    return (
      <div className="space-y-2">
        <h4 className="text-sm font-semibold text-slate-700">{title}</h4>
        <ul className="space-y-1 text-sm text-slate-600">
          {changesField.changes.map((change) => {
            const badgeClass =
              change.type === 'added'
                ? 'bg-emerald-50 text-emerald-600'
                : change.type === 'removed'
                  ? 'bg-rose-50 text-rose-600'
                  : 'bg-amber-50 text-amber-600'
            const label =
              change.type === 'added'
                ? '新增'
                : change.type === 'removed'
                  ? '删除'
                  : '修改'
            return (
              <li key={`${change.type}-${change.key}`} className="flex flex-wrap items-center gap-2">
                <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${badgeClass}`}>{label}</span>
                <span className="font-medium text-slate-700">{change.key}</span>
                {change.type !== 'added' ? <span className="text-xs text-slate-500">旧值：{change.left ?? '—'}</span> : null}
                {change.type !== 'removed' ? <span className="text-xs text-slate-500">新值：{change.right ?? '—'}</span> : null}
              </li>
            )
          })}
        </ul>
      </div>
    )
  }

  return (
    <section className="space-y-6 rounded-3xl border border-slate-200 bg-white p-8 shadow-sm">
      <header className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-900">版本历史</h2>
          <p className="text-sm text-slate-600">查看变更记录，比较差异，并将任意版本设为当前版本。</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => refetch()} disabled={isLoading}>
          刷新
        </Button>
      </header>

      {feedback ? (
        <Alert variant={feedback.type === 'success' ? 'success' : 'error'} className="text-sm">
          {feedback.message}
        </Alert>
      ) : null}

      {isLoading ? (
        <p className="text-sm text-slate-500">版本列表加载中...</p>
      ) : isError ? (
        <Alert variant="error" className="text-sm">
          {(error as Error).message || '无法加载版本列表。'}
        </Alert>
      ) : versions.length === 0 ? (
        <p className="text-sm text-slate-500">暂无版本记录，保存内容后会自动生成版本。</p>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
          <div className="space-y-2">
            <ul className="divide-y divide-slate-100 overflow-hidden rounded-2xl border border-slate-200">
              {versions.map((version) => {
                const isSelected = version.id === selectedVersionId
                const isActive = version.id === activeVersionId
                return (
                  <li
                    key={version.id}
                    className={`flex flex-col gap-3 p-4 transition hover:bg-slate-50 ${isSelected ? 'bg-slate-50' : ''}`}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-3">
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-slate-900">版本 {version.versionNumber}</span>
                          {isActive ? <Badge className="bg-brand-50 text-brand-700 ring-brand-200">当前</Badge> : null}
                        </div>
                        <p className="text-xs text-slate-500">
                          {format(new Date(version.createdAt), 'yyyy-MM-dd HH:mm')} ·{' '}
                          {version.createdBy ?? '系统'}
                        </p>
                      </div>
                      <div className="flex items-center gap-2">
                        <Button
                          type="button"
                          size="sm"
                          variant={isSelected ? 'primary' : 'secondary'}
                          onClick={() => handleSelectVersion(version.id)}
                        >
                          查看差异
                        </Button>
                        {isActive ? (
                          <span className="text-xs text-slate-500">当前版本</span>
                        ) : (
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={activateMutation.isLoading}
                            onClick={() => handleActivate(version.id)}
                          >
                            {activateMutation.isLoading && activateMutation.variables === version.id
                              ? '设为当前...'
                              : '设为当前'}
                          </Button>
                        )}
                      </div>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant="neutral">
                        {version.status === 'published' ? '已发布' : version.status === 'draft' ? '草稿' : '归档'}
                      </Badge>
                      {!isActive ? null : <span className="text-xs text-slate-400">与当前版本一致</span>}
                    </div>
                  </li>
                )
              })}
            </ul>
          </div>

          <div className="space-y-4 rounded-2xl border border-slate-200 p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <h3 className="text-lg font-semibold text-slate-900">差异对比</h3>
                <p className="text-xs text-slate-500">
                  {compareMode === 'active' ? '与当前正在使用的版本比较。' : '与上一版本比较。'}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant={compareMode === 'previous' ? 'primary' : 'secondary'}
                  onClick={() => setCompareMode('previous')}
                >
                  对上一版本
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant={compareMode === 'active' ? 'primary' : 'secondary'}
                  onClick={() => setCompareMode('active')}
                >
                  对当前版本
                </Button>
              </div>
            </div>

            {renderDiffBody()}

            <div className="space-y-4">
              {renderFieldDiff('变量 Schema 变更', diff?.variablesSchema)}
              {renderFieldDiff('Metadata 变更', diff?.metadata)}
              {!isDiffLoading && !isDiffError && diff && !diff.variablesSchema && !diff.metadata ? (
                <p className="text-sm text-slate-500">变量与 Metadata 与比较版本一致。</p>
              ) : null}
            </div>
          </div>
        </div>
      )}
    </section>
  )
}

function parseActivationError(error: unknown): string {
  const axiosError = error as AxiosError<{ message?: string }> | undefined
  if (!axiosError) {
    return '设置失败，请稍后重试。'
  }
  return axiosError.response?.data?.message ?? axiosError.message ?? '设置失败，请稍后重试。'
}
