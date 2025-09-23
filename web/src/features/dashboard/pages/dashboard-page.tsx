import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { CreatePromptModal } from '@/features/prompts/components/create-prompt-modal'
import { PromptListSection } from '@/features/prompts/components/prompt-list-section'

export function DashboardPage() {
  const [createOpen, setCreateOpen] = useState(false)

  return (
    <>
      <section className="space-y-6">
        <header className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">Prompt 管理</h1>
            <p className="mt-1 text-sm text-slate-600">
              查看所有 Prompt 模板，了解使用情况并快速进入编辑或发布流程。
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Button type="button" onClick={() => setCreateOpen(true)}>
              新建 Prompt
            </Button>
            <Button type="button" variant="secondary">
              导入模板
            </Button>
          </div>
        </header>

        <PromptListSection onCreatePrompt={() => setCreateOpen(true)} />
      </section>

      <CreatePromptModal open={createOpen} onClose={() => setCreateOpen(false)} />
    </>
  )
}
