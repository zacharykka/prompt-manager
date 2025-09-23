import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import type { Prompt } from '@/features/prompts/types'

interface DeletePromptDialogProps {
  open: boolean
  prompt: Prompt | null
  isProcessing: boolean
  errorMessage?: string | null
  onCancel: () => void
  onConfirm: () => void
}

export function DeletePromptDialog({
  open,
  prompt,
  isProcessing,
  errorMessage,
  onCancel,
  onConfirm,
}: DeletePromptDialogProps) {
  if (!open || !prompt) {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <div className="w-full max-w-lg rounded-2xl bg-white shadow-xl">
        <header className="border-b border-slate-200 px-6 py-4">
          <h2 className="text-lg font-semibold text-slate-900">确认删除 Prompt</h2>
          <p className="mt-1 text-sm text-slate-500">
            删除操作会将 Prompt 标记为“已删除”，它将从列表中移除并记录在审计日志中。
          </p>
        </header>

        <div className="space-y-4 px-6 py-5 text-sm text-slate-600">
          <p>
            即将删除的 Prompt：
            <span className="ml-1 font-medium text-slate-900">{prompt.name}</span>
          </p>
          <p className="text-xs text-slate-500">
            删除后仍可通过审计日志查看操作记录，但前端列表将不再显示该 Prompt。
          </p>

          {errorMessage ? <Alert variant="error">{errorMessage}</Alert> : null}
        </div>

        <footer className="flex items-center justify-end gap-3 border-t border-slate-200 px-6 py-4">
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={onCancel}
            disabled={isProcessing}
          >
            取消
          </Button>
          <Button
            type="button"
            variant="danger"
            size="sm"
            onClick={onConfirm}
            disabled={isProcessing}
          >
            {isProcessing ? '删除中…' : '确认删除'}
          </Button>
        </footer>
      </div>
    </div>
  )
}
