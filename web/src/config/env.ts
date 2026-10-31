/** 前端支持的部署环境。 */
export type AppEnv = 'local' | 'dev' | 'test' | 'prod'

const DEFAULT_REQUEST_TIMEOUT = 15_000

/** 移除地址末尾的斜杠，避免拼接时产生重复分隔符。 */
function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

/** 确保地址以单个斜杠结尾，供 tus 和分享前缀使用。 */
function ensureTrailingSlash(value: string): string {
  return `${trimTrailingSlash(value)}/`
}

/** 读取非空环境变量，并在缺失时使用回退值。 */
function readEnv(value: string | undefined, fallback: string): string {
  return value?.trim() || fallback
}

/** 将环境变量转换为已知环境标识。 */
function normalizeAppEnv(value: string | undefined): AppEnv {
  if (value === 'local' || value === 'dev' || value === 'test' || value === 'prod') {
    return value
  }
  return import.meta.env.PROD ? 'prod' : 'local'
}

/** 将环境变量转换为有效的请求超时时间。 */
function normalizeRequestTimeout(value: string | undefined): number {
  const timeout = Number.parseInt(value ?? '', 10)
  return Number.isFinite(timeout) && timeout > 0 ? timeout : DEFAULT_REQUEST_TIMEOUT
}

/** 前端运行时可读取的集中配置。 */
export const appConfig = Object.freeze({
  appEnv: normalizeAppEnv(import.meta.env.VITE_APP_ENV),
  adminUrl: readEnv(import.meta.env.VITE_ADMIN_URL, '/admin'),
  apiBaseUrl: trimTrailingSlash(readEnv(import.meta.env.VITE_API_BASE_URL, '/admin')),
  loginUrl: readEnv(import.meta.env.VITE_LOGIN_URL, '/admin/login'),
  tusEndpoint: ensureTrailingSlash(
    readEnv(import.meta.env.VITE_TUS_ENDPOINT, '/admin/files/tus/'),
  ),
  shareBaseUrl: ensureTrailingSlash(readEnv(import.meta.env.VITE_SHARE_BASE_URL, '/shares/')),
  requestTimeout: normalizeRequestTimeout(import.meta.env.VITE_REQUEST_TIMEOUT),
})

/** 在后台 API 根地址后拼接业务接口路径。 */
export function apiUrl(path = ''): string {
  if (/^https?:\/\//i.test(path)) {
    return path
  }

  const normalizedPath = path.replace(/^\/+/, '')
  return normalizedPath ? `${appConfig.apiBaseUrl}/${normalizedPath}` : appConfig.apiBaseUrl
}

/** 根据分享 token 生成文件下载地址。 */
export function shareDownloadUrl(token: string): string {
  const normalizedToken = encodeURIComponent(token.replace(/^\/+|\/+$/g, ''))
  return `${appConfig.shareBaseUrl}${normalizedToken}/download`
}
