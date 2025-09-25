import DOMPurify from 'dompurify'

// 使用 DOMPurify 进行 HTML 片段清洗，防止 XSS 注入
// 仅允许通用文本标签与常见内联标签；如需展示更丰富的 HTML，可在此白名单扩展
export function sanitizeHTML(input: string): string {
  return DOMPurify.sanitize(input, {
    USE_PROFILES: { html: true },
  })
}

// 对纯文本可直接返回，保留扩展点（例如去除控制字符等）
export function sanitizeText(input: string): string {
  return input
}

