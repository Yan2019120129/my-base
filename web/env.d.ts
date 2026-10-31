/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** 当前部署环境标识。 */
  readonly VITE_APP_ENV?: 'local' | 'dev' | 'test' | 'prod'

  /** 工作台入口地址，未设置时使用同源 /admin。 */
  readonly VITE_ADMIN_URL?: string

  /** 前台站点基础地址，不带结尾斜杠。 */
  readonly VITE_SITE_URL?: string

  /** 后台 API 根地址，包含后台前缀 /admin。 */
  readonly VITE_API_BASE_URL?: string

  /** 后台登录地址。 */
  readonly VITE_LOGIN_URL?: string

  /** tus 断点续传地址，必须以斜杠结尾。 */
  readonly VITE_TUS_ENDPOINT?: string

  /** 文件分享地址前缀。 */
  readonly VITE_SHARE_BASE_URL?: string

  /** 请求超时时间，单位为毫秒。 */
  readonly VITE_REQUEST_TIMEOUT?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
