import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { AxiosError } from 'axios'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { createPrompt } from '@/features/prompts/api/create-prompt'

const createPromptSchema = z.object({
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
})

type CreatePromptValues = z.infer<typeof createPromptSchema>

interface CreatePromptModalProps {
  open: boolean
  onClose: () => void
}

export function CreatePromptModal({ open, onClose }: CreatePromptModalProps) {
  const queryClient = useQueryClient()
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreatePromptValues>({
    resolver: zodResolver(createPromptSchema),
    defaultValues: {
      name: '',
      description: '',
      tags: '',
    },
  })

  const {
    mutateAsync,
    isPending,
    isError,
    error,
  } = useMutation({
    mutationFn: async (values: CreatePromptValues) => {
      const payload = {
        name: values.name.trim(),
        description: values.description?.trim() || undefined,
        tags: normalizeTags(values.tags),
      }
      return createPrompt(payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['prompts'] })
      onClose()
      reset({ name: '', description: '', tags: '' })
    },
  })

  useEffect(() => {
    if (!open) {
      reset({ name: '', description: '', tags: '' })
    }
  }, [open, reset])

  const onSubmit = async (values: CreatePromptValues) => {
    await mutateAsync(values)
  }

  if (!open) {
    return null
  }

  const axiosError = error as AxiosError<{ message?: string }> | undefined
  const errorMessage = axiosError?.response?.data?.message ?? axiosError?.message

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <div className="w-full max-w-2xl rounded-2xl bg-white shadow-xl">
        <header className="flex items-start justify-between border-b border-slate-200 px-6 py-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">新建 Prompt</h2>
            <p className="mt-1 text-sm text-slate-500">
              输入 Prompt 基本信息，可稍后在详情页补充变量与版本。
            </p>
          </div>
          <Button variant="ghost" onClick={onClose} type="button">
            关闭
          </Button>
        </header>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-5 px-6 py-5">
          {isError && errorMessage ? (
            <Alert variant="error">{errorMessage}</Alert>
          ) : null}

          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700" htmlFor="prompt-name">
              名称
            </label>
            <Input id="prompt-name" placeholder="例如：欢迎语" {...register('name')} />
            {errors.name ? (
              <p className="text-xs text-red-600">{errors.name.message}</p>
            ) : null}
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700" htmlFor="prompt-description">
              描述（可选）
            </label>
            <Textarea
              id="prompt-description"
              placeholder="简要描述用途，方便团队快速理解"
              rows={4}
              {...register('description')}
            />
            {errors.description ? (
              <p className="text-xs text-red-600">{errors.description.message}</p>
            ) : null}
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-slate-700" htmlFor="prompt-tags">
              标签（逗号分隔，可选）
            </label>
            <Input
              id="prompt-tags"
              placeholder="例如：欢迎, 客服, 多语言"
              {...register('tags')}
            />
            {errors.tags ? (
              <p className="text-xs text-red-600">{errors.tags.message}</p>
            ) : null}
          </div>

          <div className="flex items-center justify-end gap-3 border-t border-slate-200 pt-4">
            <Button type="button" variant="secondary" onClick={onClose} disabled={isPending}>
              取消
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? '创建中...' : '创建 Prompt'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}

function normalizeTags(value: string | undefined): string[] | undefined {
  if (!value) {
    return undefined
  }
  const parts = value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)

  return parts.length > 0 ? Array.from(new Set(parts)) : undefined
}
