import { format } from 'date-fns'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { Prompt } from '@/features/prompts/types'

interface PromptTableProps {
  prompts: Prompt[]
  onDeletePrompt?: (prompt: Prompt) => void
  deletingPromptId?: string | null
  disableActions?: boolean
  onEditPrompt?: (prompt: Prompt) => void
}

export function PromptTable({
  prompts,
  onDeletePrompt,
  deletingPromptId,
  disableActions = false,
  onEditPrompt,
}: PromptTableProps) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <table className="min-w-full divide-y divide-slate-200">
        <thead className="bg-slate-50">
          <tr className="text-left text-sm font-semibold text-slate-600">
            <th scope="col" className="px-4 py-3">名称</th>
            <th scope="col" className="px-4 py-3">内容</th>
            <th scope="col" className="px-4 py-3">标签</th>
            <th scope="col" className="px-4 py-3">状态</th>
            <th scope="col" className="px-4 py-3">更新人</th>
            <th scope="col" className="px-4 py-3">更新时间</th>
            {onDeletePrompt ? (
              <th scope="col" className="px-4 py-3 text-right">操作</th>
            ) : null}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {prompts.map((prompt) => (
            <tr key={prompt.id} className="text-sm text-slate-700">
              <td className="max-w-xs px-4 py-3">
                <div className="flex flex-col gap-1">
                  {onEditPrompt ? (
                    <button
                      type="button"
                      onClick={() => onEditPrompt(prompt)}
                      className="truncate text-left font-medium text-brand-600 hover:underline"
                    >
                      {prompt.name}
                    </button>
                  ) : (
                    <span className="truncate font-medium text-slate-900">{prompt.name}</span>
                  )}
                  {prompt.description ? (
                    <span className="truncate text-xs text-slate-500">
                      {prompt.description}
                    </span>
                  ) : null}
                </div>
              </td>
              <td className="max-w-md px-4 py-3">
                {prompt.body ? (
                  <p className="line-clamp-2 text-xs text-slate-600">{prompt.body}</p>
                ) : (
                  <span className="text-xs text-slate-400">暂无内容</span>
                )}
              </td>
              <td className="px-4 py-3">
                <div className="flex flex-wrap gap-2">
                  {prompt.tags.length > 0 ? (
                    prompt.tags.map((tag) => (
                      <Badge key={tag} variant="neutral">
                        {tag}
                      </Badge>
                    ))
                  ) : (
                    <span className="text-xs text-slate-400">未设置</span>
                  )}
                </div>
              </td>
              <td className="px-4 py-3">
                {prompt.status === 'deleted' ? (
                  <Badge variant="danger">已删除</Badge>
                ) : prompt.activeVersionId ? (
                  <Badge variant="default">已发布</Badge>
                ) : (
                  <Badge variant="neutral">草稿</Badge>
                )}
              </td>
              <td className="px-4 py-3 text-xs text-slate-500">
                {prompt.createdBy ?? '系统'}
              </td>
              <td className="px-4 py-3 text-xs text-slate-500">
                {format(new Date(prompt.updatedAt), 'yyyy-MM-dd HH:mm')}
              </td>
              {onDeletePrompt ? (
                <td className="px-4 py-3 text-right">
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => onDeletePrompt(prompt)}
                    disabled={disableActions || deletingPromptId === prompt.id}
                  >
                    {deletingPromptId === prompt.id ? '删除中...' : '删除'}
                  </Button>
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
