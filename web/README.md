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

## 打包 Web

所有打包命令都在 `web` 目录执行。打包流程会安装锁定依赖，并依次执行 lint、类型检查、构建和发布包校验。

```sh
cd web

make        # 生产环境
make prod   # 生产环境
make dev    # 开发环境
make test   # 测试环境
```

发布包输出到 `build/`，文件名中包含 12 位时间戳和压缩包的完整 SHA-256，不再生成独立的 `.sha256` 文件：

```text
dist.<pro|dev|test>.<YYYYMMDDHHMM>.<64位SHA256>.zip
```

例如 `dist.pro.202611010514.<SHA256>.zip`。ZIP 根目录直接包含 `index.html` 和 `assets/`，可以解压到任意部署目录。

查看并解压最新的生产环境发布包：

```sh
cd web

ZIP_FILE=$(find build -maxdepth 1 -type f -name 'dist.pro.*.zip' \
  -printf '%T@ %p\n' | sort -nr | head -n 1 | cut -d' ' -f2-)
ZIP_NAME=$(basename "$ZIP_FILE" .zip)
PACKAGE_NAME=${ZIP_NAME%.*}

unzip -l "$ZIP_FILE"
unzip -o "$ZIP_FILE" -d "build/$PACKAGE_NAME"
```

解压后的文件位于 `build/$PACKAGE_NAME/`，该目录根部就是 `index.html` 和 `assets/`。

根据文件名中的 SHA-256 校验压缩包：

```sh
ZIP_FILE=$(find build -maxdepth 1 -type f -name 'dist.pro.*.zip' \
  -printf '%T@ %p\n' | sort -nr | head -n 1 | cut -d' ' -f2-)
ZIP_NAME=$(basename "$ZIP_FILE" .zip)
EXPECTED_SHA256=${ZIP_NAME##*.}
ACTUAL_SHA256=$(sha256sum "$ZIP_FILE" | awk '{print $1}')

test "$EXPECTED_SHA256" = "$ACTUAL_SHA256" \
  && echo "SHA-256 校验通过"
```

## 环境变量

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `VITE_APP_ENV` | 环境标识：`local`、`dev`、`test`、`prod` | 开发模式为 `local`，生产构建为 `prod` |
| `VITE_ADMIN_URL` | 后台工作台入口 | `/admin` |
| `VITE_SITE_URL` | 首页 canonical、`og:url` 和 `og:image` 使用的站点基础地址，不带结尾斜杠 | 无 |
| `VITE_API_BASE_URL` | 后台 API 根地址，包含 `/admin` | `/admin` |
| `VITE_LOGIN_URL` | 后台登录入口 | `/admin/login` |
| `VITE_TUS_ENDPOINT` | tus 断点续传地址 | `/admin/files/tus/` |
| `VITE_SHARE_BASE_URL` | 文件分享地址前缀 | `/shares/` |
| `VITE_REQUEST_TIMEOUT` | 请求超时毫秒数 | `15000` |

完整模板见 `.env.example`。API 调用使用 `src/api/http.ts` 的 `apiRequest`，默认携带登录 Cookie 并处理超时和错误响应。

`VITE_SITE_URL` 在构建时写入 `index.html`，因此不同环境需要使用对应的构建命令。本机默认地址为 `http://localhost:5173`；如果 Vite 自动切换端口，请同步修改 `.env.local` 中的该变量。

> 当前后台尚未配置 CORS。前端跨域请求后台 API 前，需要后台允许对应环境的前端来源并支持携带凭据的请求。
