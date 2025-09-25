import { useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { format } from 'date-fns'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { AxiosError } from 'axios'

import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { activatePromptVersion } from '@/features/prompts/api/activate-prompt-version'
import { usePromptVersionDiff, usePromptVersionsQuery } from '@/features/prompts/hooks/use-prompt-versions'
import type { PromptVersion, PromptVersionDiff, PromptVersionDiffField } from '@/features/prompts/types'

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
  const [diffVersionId, setDiffVersionId] = useState<string | null>(null)
  const [isDiffDialogOpen, setDiffDialogOpen] = useState(false)
  const [compareMode, setCompareMode] = useState<CompareMode>('previous')
  const [feedback, setFeedback] = useState<FeedbackState>(null)

  const { data, isLoading, isError, error, refetch } = usePromptVersionsQuery(promptId)
  const versionItems = data?.items
  const versions = useMemo(() => versionItems ?? [], [versionItems])
  const defaultVersionId = useMemo(() => {
    if (!versions.length) {
      return null
    }
    const active = versions.find((item) => item.id === activeVersionId)
    return active?.id ?? versions[0].id
  }, [versions, activeVersionId])

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
  } = usePromptVersionDiff(promptId, diffVersionId, diffParams)

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

  const handleOpenDiff = (versionId: string) => {
    setSelectedVersionId(versionId)
    setDiffVersionId(versionId)
    setCompareMode('previous')
    setDiffDialogOpen(true)
  }

  const handleCloseDiff = () => {
    setDiffDialogOpen(false)
    setDiffVersionId(null)
    setCompareMode('previous')
  }

  const diffVersion = useMemo(() => {
    if (!diffVersionId) {
      return null
    }
    return versions.find((item) => item.id === diffVersionId) ?? null
  }, [versions, diffVersionId])

  return (
    <>
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
          <ul className="divide-y divide-slate-100 overflow-hidden rounded-2xl border border-slate-200 bg-white">
            {versions.map((version) => {
              const isSelected = version.id === selectedVersionId
              const isActive = version.id === activeVersionId
              return (
                <li
                  key={version.id}
                  className={`flex flex-col gap-3 p-4 transition ${
                    isSelected
                      ? 'bg-brand-50/40 ring-1 ring-inset ring-brand-200'
                      : 'bg-white hover:bg-slate-50'
                  }`}
                >
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-slate-900">版本 {version.versionNumber}</span>
                        {isActive ? <Badge className="bg-brand-50 text-brand-700 ring-brand-200">当前</Badge> : null}
                      </div>
                      <p className="text-xs text-slate-500">
                        {formatDateTime(version.createdAt)} · {version.createdBy ?? '系统'}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        type="button"
                        size="sm"
                        variant={isSelected ? 'primary' : 'secondary'}
                        onClick={() => handleOpenDiff(version.id)}
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
        )}
      </section>

      <PromptVersionDiffDialog
        open={isDiffDialogOpen}
        version={diffVersion}
        diff={diff}
        compareMode={compareMode}
        isLoading={isDiffLoading}
        isError={isDiffError}
        error={diffError}
        onCompareModeChange={setCompareMode}
        onClose={handleCloseDiff}
        promptName={promptName}
      />
    </>
  )
}

function parseActivationError(error: unknown): string {
  const axiosError = error as AxiosError<{ message?: string }> | undefined
  if (!axiosError) {
    return '设置失败，请稍后重试。'
  }
  return axiosError.response?.data?.message ?? axiosError.message ?? '设置失败，请稍后重试。'
}

function formatDateTime(value?: string | null): string {
  if (!value) {
    return '—'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '—'
  }
  return format(date, 'yyyy-MM-dd HH:mm')
}

interface PromptVersionDiffDialogProps {
  open: boolean
  version: PromptVersion | null
  diff: PromptVersionDiff | undefined
  compareMode: CompareMode
  isLoading: boolean
  isError: boolean
  error: unknown
  onCompareModeChange: (mode: CompareMode) => void
  onClose: () => void
  promptName?: string
}

function PromptVersionDiffDialog({
  open,
  version,
  diff,
  compareMode,
  isLoading,
  isError,
  error,
  onCompareModeChange,
  onClose,
  promptName,
}: PromptVersionDiffDialogProps) {
  if (!open || !version) {
    return null
  }

  const renderDiffBody = () => {
    if (isLoading) {
      return <p className="text-sm text-slate-500">差异计算中...</p>
    }

    if (isError) {
      const axiosError = error as AxiosError<{ message?: string }> | undefined
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
      <div className="rounded-xl bg-slate-950 p-4 font-mono text-xs text-slate-100 whitespace-pre-wrap overflow-x-auto">
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

  const renderFieldDiff = (title: string, changesField?: PromptVersionDiffField) => {
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

  const modalContent = (
    <div
      className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/50 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      onClick={onClose}
      style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 99999 }}
    >
      <div
        className="relative m-6 flex w-full max-w-4xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl max-h-[90vh] border border-slate-200"
        onClick={(event) => event.stopPropagation()}
      >
        <header className="flex flex-col gap-2 border-b border-slate-200 bg-white px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">差异对比</h2>
            <p className="text-xs text-slate-500">
              {promptName ? `Prompt：${promptName} · ` : ''}当前查看版本 {version.versionNumber}（
              {formatDateTime(version.createdAt)}）
            </p>
          </div>
          <Button type="button" variant="ghost" size="sm" onClick={onClose}>
            关闭
          </Button>
        </header>

        <div className="flex-1 space-y-4 overflow-y-auto px-6 py-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-xs text-slate-500">
              {compareMode === 'active' ? '与当前正在使用的版本比较。' : '与上一版本比较。'}
            </p>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                size="sm"
                variant={compareMode === 'previous' ? 'primary' : 'secondary'}
                onClick={() => onCompareModeChange('previous')}
              >
                对上一版本
              </Button>
              <Button
                type="button"
                size="sm"
                variant={compareMode === 'active' ? 'primary' : 'secondary'}
                onClick={() => onCompareModeChange('active')}
              >
                对当前版本
              </Button>
            </div>
          </div>

          {diff ? (
            <div className="rounded-xl bg-slate-50 p-4 text-xs text-slate-600">
              <p>
                目标版本：版本 {diff.target.versionNumber} · {formatDateTime(diff.target.createdAt)} ·{' '}
                {diff.target.createdBy ?? '系统'}
              </p>
              <p>
                比较版本：版本 {diff.base.versionNumber} · {formatDateTime(diff.base.createdAt)} ·{' '}
                {diff.base.createdBy ?? '系统'}
              </p>
            </div>
          ) : null}

          {renderDiffBody()}

          <div className="space-y-4">
            {renderFieldDiff('变量 Schema 变更', diff?.variablesSchema)}
            {renderFieldDiff('Metadata 变更', diff?.metadata)}
            {!isLoading && !isError && diff && !diff.variablesSchema && !diff.metadata ? (
              <p className="text-sm text-slate-500">变量与 Metadata 与比较版本一致。</p>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  )

  return open && version ? createPortal(modalContent, document.body) : null
}
