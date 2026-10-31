import { apiUrl, appConfig } from '@/config/env'

/** 后台接口返回的错误数据结构。 */
export interface ApiErrorBody {
  code?: number
  msg?: string
  message?: string
  [key: string]: unknown
}

/** 后台请求失败时抛出的统一错误。 */
export class ApiError extends Error {
  /** HTTP 状态码。 */
  readonly status: number

  /** 后台返回的原始错误内容。 */
  readonly body: unknown

  constructor(status: number, body: unknown) {
    super(readErrorMessage(body) ?? `API request failed with status ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

/** API 请求选项，可在标准 RequestInit 基础上覆盖超时时间。 */
export interface ApiRequestOptions extends RequestInit {
  /** 本次请求的超时时间，单位为毫秒。 */
  timeoutMs?: number
}

/** 从常见错误响应中提取可展示的信息。 */
function readErrorMessage(body: unknown): string | undefined {
  if (typeof body === 'string') {
    return body.trim() || undefined
  }
  if (!body || typeof body !== 'object') {
    return undefined
  }

  const record = body as Record<string, unknown>
  for (const key of ['msg', 'message', 'error']) {
    const value = record[key]
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
  }
  return undefined
}

/** 根据响应类型读取 JSON 或文本内容。 */
async function readResponseBody(response: Response): Promise<unknown> {
  const contentType = response.headers.get('content-type') ?? ''
  if (contentType.includes('application/json')) {
    return response.json()
  }

  const text = await response.text()
  return text || null
}

/**
 * 发起带后台登录 Cookie 的 JSON 请求。
 *
 * 调用方传入 /files/... 等业务路径时，会自动拼接 API 根地址。
 */
export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
  const {
    timeoutMs = appConfig.requestTimeout,
    signal,
    headers,
    ...requestInit
  } = options
  const controller = new AbortController()
  const abortFromCaller = () => controller.abort(signal?.reason)

  if (signal?.aborted) {
    controller.abort(signal.reason)
  } else {
    signal?.addEventListener('abort', abortFromCaller, { once: true })
  }

  const requestHeaders = new Headers(headers)
  if (!requestHeaders.has('Accept')) {
    requestHeaders.set('Accept', 'application/json')
  }

  const timeout = window.setTimeout(() => controller.abort(), timeoutMs)

  try {
    const response = await fetch(apiUrl(path), {
      ...requestInit,
      credentials: requestInit.credentials ?? 'include',
      headers: requestHeaders,
      signal: controller.signal,
    })

    if (response.status === 204) {
      return undefined as T
    }

    const body = await readResponseBody(response)
    if (!response.ok) {
      throw new ApiError(response.status, body)
    }
    return body as T
  } finally {
    window.clearTimeout(timeout)
    signal?.removeEventListener('abort', abortFromCaller)
  }
}
