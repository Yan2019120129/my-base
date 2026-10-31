# 020200 Web

020200 个人网站首页，使用 Vue 3、TypeScript 和 Vite 构建。项目内的页面、样式与图片资源均位于 `web` 目录，不依赖仓库中的其他前端文件。

## 环境要求

- Node.js `^22.18.0` 或 `>=24.12.0`
- pnpm

## 环境文件

项目内置四套可提交的环境配置：

| 文件 | 用途 | 后台地址 |
| --- | --- | --- |
| `.env.development` | 本机开发 | `http://localhost:1050` |
| `.env.dev` | 远程开发 | `https://dev.020200.xyz` |
| `.env.test` | 测试 | `https://test.020200.xyz` |
| `.env.production` | 生产 | `https://020200.xyz` |

如需临时覆盖配置，创建被 Git 忽略的 `.env.local`。它的优先级高于所有环境文件。

## 本地开发

```sh
pnpm install
pnpm dev
```

`pnpm dev` 默认使用本机后台。连接远程开发环境时运行：

```sh
pnpm dev:dev
```

## 构建与检查

```sh
pnpm type-check
pnpm lint
pnpm build
pnpm build:dev
pnpm build:test
```

`pnpm build` 使用生产配置，`pnpm build:dev` 和 `pnpm build:test` 分别使用远程开发与测试配置。

## 环境变量

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `VITE_APP_ENV` | 环境标识：`local`、`dev`、`test`、`prod` | 开发模式为 `local`，生产构建为 `prod` |
| `VITE_ADMIN_URL` | 后台工作台入口 | `/admin` |
| `VITE_API_BASE_URL` | 后台 API 根地址，包含 `/admin` | `/admin` |
| `VITE_LOGIN_URL` | 后台登录入口 | `/admin/login` |
| `VITE_TUS_ENDPOINT` | tus 断点续传地址 | `/admin/files/tus/` |
| `VITE_SHARE_BASE_URL` | 文件分享地址前缀 | `/shares/` |
| `VITE_REQUEST_TIMEOUT` | 请求超时毫秒数 | `15000` |

完整模板见 `.env.example`。API 调用使用 `src/api/http.ts` 的 `apiRequest`，默认携带登录 Cookie 并处理超时和错误响应。

> 当前后台尚未配置 CORS。前端跨域请求后台 API 前，需要后台允许对应环境的前端来源并支持携带凭据的请求。
