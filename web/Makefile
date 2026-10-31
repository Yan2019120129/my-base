PNPM ?= pnpm
WEB_ENV ?= prod
BUILD_ROOT := build
TIMESTAMP := $(shell date +%Y%m%d%H%M)
PACKAGE_ENV = $(if $(filter prod,$(WEB_ENV)),pro,$(WEB_ENV))
PACKAGE_PREFIX = dist.$(PACKAGE_ENV)
BUILD_MODE = $(if $(filter prod,$(WEB_ENV)),production,$(WEB_ENV))
TEMP_ZIP = $(BUILD_ROOT)/.$(PACKAGE_PREFIX).$(TIMESTAMP).zip.tmp

.PHONY: all prod dev test package

all: prod

prod: WEB_ENV := prod
prod: package

dev: WEB_ENV := dev
dev: package

test: WEB_ENV := test
test: package

# 执行发布前检查、构建并生成文件名带完整 SHA-256 的 ZIP 发布包
package:
	@case "$(WEB_ENV)" in \
		dev|test|prod) ;; \
		*) echo "❌ WEB_ENV 仅支持 dev、test、prod"; exit 1 ;; \
	esac
	@command -v $(PNPM) >/dev/null 2>&1 || { echo "❌ 未找到 $(PNPM)，请先安装 pnpm"; exit 1; }
	@command -v zip >/dev/null 2>&1 || { echo "❌ 未找到 zip 命令"; exit 1; }
	@command -v unzip >/dev/null 2>&1 || { echo "❌ 未找到 unzip 命令"; exit 1; }
	@command -v sha256sum >/dev/null 2>&1 || { echo "❌ 未找到 sha256sum 命令"; exit 1; }
	@command -v awk >/dev/null 2>&1 || { echo "❌ 未找到 awk 命令"; exit 1; }
	@echo "🚀 开始 Web 发布检查与构建（$(WEB_ENV)）..."
	@$(PNPM) install --frozen-lockfile --prefer-offline
	@$(PNPM) exec oxlint .
	@$(PNPM) exec eslint . --cache
	@$(PNPM) type-check
	@$(PNPM) build-only --mode "$(BUILD_MODE)"
	@test -f dist/index.html || { echo "❌ 构建产物缺少 dist/index.html"; exit 1; }
	@test -d dist/assets || { echo "❌ 构建产物缺少 dist/assets/"; exit 1; }
	@mkdir -p "$(BUILD_ROOT)"
	@rm -f "$(TEMP_ZIP)"
	@(cd dist && zip -qr "$(abspath $(TEMP_ZIP))" .)
	@set -eu; \
		SHA256=$$(sha256sum "$(TEMP_ZIP)" | awk '{print $$1}'); \
		ZIP_FILE="$(BUILD_ROOT)/$(PACKAGE_PREFIX).$(TIMESTAMP).$$SHA256.zip"; \
		mv -f "$(TEMP_ZIP)" "$$ZIP_FILE"; \
		unzip -t "$$ZIP_FILE" >/dev/null; \
		unzip -Z1 "$$ZIP_FILE" | grep -qx 'index.html'; \
		unzip -Z1 "$$ZIP_FILE" | grep -q '^assets/'; \
		EXPECTED_SHA256=$$(basename "$$ZIP_FILE" .zip); \
		EXPECTED_SHA256=$${EXPECTED_SHA256##*.}; \
		ACTUAL_SHA256=$$(sha256sum "$$ZIP_FILE" | awk '{print $$1}'); \
		[ "$$EXPECTED_SHA256" = "$$ACTUAL_SHA256" ]; \
		echo "✅ Web 打包完成"; \
		echo "📦 压缩包: $$ZIP_FILE"; \
		echo "🔐 SHA-256: $$ACTUAL_SHA256"
