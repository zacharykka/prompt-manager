import { format } from 'date-fns'

import { Badge } from '@/components/ui/badge'
import type { Prompt } from '@/features/prompts/types'

interface PromptTableProps {
  prompts: Prompt[]
}

export function PromptTable({ prompts }: PromptTableProps) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <table className="min-w-full divide-y divide-slate-200">
        <thead className="bg-slate-50">
          <tr className="text-left text-sm font-semibold text-slate-600">
            <th scope="col" className="px-4 py-3">名称</th>
            <th scope="col" className="px-4 py-3">标签</th>
            <th scope="col" className="px-4 py-3">状态</th>
            <th scope="col" className="px-4 py-3">更新人</th>
            <th scope="col" className="px-4 py-3">更新时间</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {prompts.map((prompt) => (
            <tr key={prompt.id} className="text-sm text-slate-700">
              <td className="max-w-xs px-4 py-3">
                <div className="flex flex-col gap-1">
                  <span className="truncate font-medium text-slate-900">{prompt.name}</span>
                  {prompt.description ? (
                    <span className="truncate text-xs text-slate-500">
                      {prompt.description}
                    </span>
                  ) : null}
                </div>
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
                {prompt.activeVersionId ? (
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
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
