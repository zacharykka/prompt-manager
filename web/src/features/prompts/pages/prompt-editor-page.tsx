import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { AxiosError } from 'axios'
import { useQueryClient } from '@tanstack/react-query'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { createPrompt } from '@/features/prompts/api/create-prompt'
import { createPromptVersion } from '@/features/prompts/api/create-prompt-version'
import { updatePrompt } from '@/features/prompts/api/update-prompt'
import { usePromptQuery } from '@/features/prompts/hooks/use-prompt'
import { PromptVersionPanel } from '@/features/prompts/components/prompt-version-panel'
import type { Prompt } from '@/features/prompts/types'

const promptEditorSchema = z.object({
  name: z
    .string()
    .min(1, '名称不能为空')
    .max(128, '名称长度最大 128 字符'),
  description: z
    .string()
    .max(500, '描述长度最大 500 字符')
    .optional()
    .or(z.literal('')),
  tags: z
    .string()
    .optional()
    .or(z.literal('')),
  body: z
    .string()
    .min(1, '内容不能为空'),
})

type PromptEditorValues = z.infer<typeof promptEditorSchema>

type EditorMode = 'create' | 'edit'

export function PromptEditorPage() {
  const { promptId } = useParams<{ promptId: string }>()
  const mode: EditorMode = promptId ? 'edit' : 'create'
  const navigate = useNavigate()
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [originalPrompt, setOriginalPrompt] = useState<Prompt | null>(null)
  const queryClient = useQueryClient()

  const {
    data: prompt,
    isLoading: isPromptLoading,
    isError: isPromptError,
    error: promptError,
  } = usePromptQuery(mode === 'edit' ? promptId : undefined)

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<PromptEditorValues>({
    resolver: zodResolver(promptEditorSchema),
    defaultValues: {
      name: '',
      description: '',
      tags: '',
      body: '',
    },
  })

  useEffect(() => {
    if (mode === 'edit' && prompt) {
      reset({
        name: prompt.name,
        description: prompt.description ?? '',
        tags: prompt.tags.join(', '),
        body: prompt.body ?? '',
      })
      setOriginalPrompt(prompt)
    }
  }, [mode, prompt, reset])

  const pageTitle = mode === 'create' ? '新建 Prompt' : '编辑 Prompt'

  const handleCancel = () => {
    navigate('/prompts')
  }

  const submitHandler = handleSubmit(async (values) => {
    setSubmitError(null)
    const payload = {
      name: values.name.trim(),
      description: values.description?.trim() || undefined,
      tags: normalizeTags(values.tags),
      body: values.body.trim(),
    }

    try {
      if (mode === 'create') {
        await createPrompt(payload)
        await queryClient.invalidateQueries({ queryKey: ['prompts'] })
        navigate('/prompts', {
          state: { feedback: { type: 'success', message: `Prompt “${payload.name}” 创建成功。` } },
        })
        return
      }

      if (!promptId) {
        throw new Error('缺少 Prompt ID')
      }

      await updatePrompt(promptId, {
        name: payload.name,
        description: payload.description ?? null,
        tags: payload.tags,
      })

      const originalBody = originalPrompt?.body ?? ''
      if (payload.body !== originalBody) {
        await createPromptVersion(promptId, {
          body: payload.body,
          status: 'published',
          activate: true,
        })
      }

      await queryClient.invalidateQueries({ queryKey: ['prompts'] })
      await queryClient.invalidateQueries({ queryKey: ['prompt', promptId] })
      await queryClient.invalidateQueries({ queryKey: ['promptVersions', promptId] })
      navigate('/prompts', {
        state: { feedback: { type: 'success', message: `Prompt “${payload.name}” 已更新。` } },
      })
    } catch (error) {
      const message = parseSubmitError(error)
      setSubmitError(message)
    }
  })

  const promptLoadingOrError = useMemo(() => {
    if (mode === 'create') {
      return null
    }
    if (isPromptLoading) {
      return 'loading'
    }
    if (isPromptError) {
      return 'error'
    }
    return null
  }, [mode, isPromptLoading, isPromptError])

  if (promptLoadingOrError === 'loading') {
    return (
      <div className="mx-auto max-w-4xl space-y-6">
        <header className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">加载 Prompt...</h1>
            <p className="mt-1 text-sm text-slate-600">请稍候，正在获取 Prompt 详情。</p>
          </div>
        </header>
      </div>
    )
  }

  if (promptLoadingOrError === 'error') {
    const axiosError = promptError as AxiosError<{ message?: string }> | undefined
    const message =
      axiosError?.response?.data?.message ??
      axiosError?.message ??
      '未能加载指定的 Prompt，请返回重试。'

    return (
      <div className="mx-auto max-w-4xl space-y-6">
        <header className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-semibold text-slate-900">加载失败</h1>
            <p className="mt-1 text-sm text-slate-600">{message}</p>
          </div>
          <Button type="button" variant="secondary" onClick={handleCancel}>
            返回列表
          </Button>
        </header>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-screen-xl space-y-10 px-6 py-4">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-slate-900">{pageTitle}</h1>
          <p className="mt-1 text-sm text-slate-600">
            {mode === 'create'
              ? '填写 Prompt 详细信息并创建首个版本。'
              : '更新 Prompt 的基础信息，保存后自动激活最新版本。'}
          </p>
        </div>
        <Button type="button" variant="secondary" onClick={handleCancel}>
          返回列表
        </Button>
      </header>

      <form onSubmit={submitHandler} className="space-y-10 rounded-3xl border border-slate-200 bg-white p-12 shadow-md">
        {submitError ? <Alert variant="error">{submitError}</Alert> : null}

        <div className="grid gap-8 lg:grid-cols-3">
          <div className="space-y-2 lg:col-span-2">
            <label className="block text-sm font-medium text-slate-700" htmlFor="prompt-name">
              名称
            </label>
            <Input id="prompt-name" placeholder="例如：欢迎语" {...register('name')} />
            {errors.name ? <p className="text-xs text-red-600">{errors.name.message}</p> : null}
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700" htmlFor="prompt-tags">
              标签（逗号分隔，可选）
            </label>
            <Input id="prompt-tags" placeholder="例如：欢迎, 客服" {...register('tags')} />
            {errors.tags ? <p className="text-xs text-red-600">{errors.tags.message}</p> : null}
          </div>
        </div>

        <div className="space-y-2">
          <label className="block text-sm font-medium text-slate-700" htmlFor="prompt-description">
            描述（可选）
          </label>
          <Input
            id="prompt-description"
            placeholder="简要描述用途，方便团队快速理解"
            {...register('description')}
          />
          {errors.description ? (
            <p className="text-xs text-red-600">{errors.description.message}</p>
          ) : null}
        </div>

        <div className="space-y-2">
          <label className="block text-sm font-medium text-slate-700" htmlFor="prompt-body">
            内容
          </label>
          <Textarea
            id="prompt-body"
            placeholder="编写 Prompt 正文，支持模板变量"
            rows={22}
            {...register('body')}
          />
          {errors.body ? <p className="text-xs text-red-600">{errors.body.message}</p> : null}
        </div>

        <div className="flex items-center justify-end gap-3">
          <Button type="button" variant="secondary" onClick={handleCancel} disabled={isSubmitting}>
            取消
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? '保存中...' : mode === 'create' ? '创建 Prompt' : '保存修改'}
          </Button>
        </div>
      </form>

      {mode === 'edit' && promptId ? (
        <PromptVersionPanel
          promptId={promptId}
          activeVersionId={prompt?.activeVersionId ?? null}
          promptName={prompt?.name}
        />
      ) : null}
    </div>
  )
}

function normalizeTags(value: string | undefined): string[] | undefined {
  if (!value) {
    return undefined
  }
  const normalized = value.replace(/[，、]/g, ',')
  const parts = normalized
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
  return parts.length > 0 ? Array.from(new Set(parts)) : undefined
}

function parseSubmitError(error: unknown): string {
  const axiosError = error as AxiosError<{ message?: string }> | undefined
  if (!axiosError) {
    return '提交失败，请稍后再试。'
  }
  return (
    axiosError.response?.data?.message ??
    axiosError.message ??
    '提交失败，请稍后再试。'
  )
}
