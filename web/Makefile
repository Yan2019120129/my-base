PNPM ?= pnpm
WEB_ENV ?= prod
BUILD_ROOT := build
TIMESTAMP := $(shell date +%Y%m%d%H%M)
PACKAGE_ENV = $(if $(filter prod,$(WEB_ENV)),pro,$(WEB_ENV))
PACKAGE_PREFIX = dist.$(PACKAGE_ENV)
BUILD_MODE = $(if $(filter prod,$(WEB_ENV)),production,$(WEB_ENV))
TEMP_ZIP = $(BUILD_ROOT)/.$(PACKAGE_PREFIX).$(TIMESTAMP).zip.tmp

.PHONY: all prod dev test package unpack unpack-dev unpack-test _unpack

all: prod

prod: WEB_ENV := prod
prod: package

dev: WEB_ENV := dev
dev: package

test: WEB_ENV := test
test: package

unpack: WEB_ENV := prod
unpack: _unpack

unpack-dev: WEB_ENV := dev
unpack-dev: _unpack

unpack-test: WEB_ENV := test
unpack-test: _unpack

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
		unzip -t "$(TEMP_ZIP)" >/dev/null; \
		unzip -Z1 "$(TEMP_ZIP)" | grep -qx 'index.html'; \
		unzip -Z1 "$(TEMP_ZIP)" | grep -q '^assets/'; \
		mv -f "$(TEMP_ZIP)" "$$ZIP_FILE"; \
		for OLD_ZIP in "$(BUILD_ROOT)/$(PACKAGE_PREFIX)."*.zip; do \
			[ -f "$$OLD_ZIP" ] || continue; \
			[ "$$OLD_ZIP" = "$$ZIP_FILE" ] || rm -f "$$OLD_ZIP"; \
		done; \
		EXPECTED_SHA256=$$(basename "$$ZIP_FILE" .zip); \
		EXPECTED_SHA256=$${EXPECTED_SHA256##*.}; \
		ACTUAL_SHA256=$$(sha256sum "$$ZIP_FILE" | awk '{print $$1}'); \
		[ "$$EXPECTED_SHA256" = "$$ACTUAL_SHA256" ]; \
		echo "✅ Web 打包完成"; \
		echo "📦 压缩包: $$ZIP_FILE"; \
		echo "🔐 SHA-256: $$ACTUAL_SHA256"

# 校验并解压指定环境最新发布包到与 ZIP 同名的目录
_unpack:
	@case "$(WEB_ENV)" in \
		dev|test|prod) ;; \
		*) echo "❌ WEB_ENV 仅支持 dev、test、prod"; exit 1 ;; \
	esac
	@command -v unzip >/dev/null 2>&1 || { echo "❌ 未找到 unzip 命令"; exit 1; }
	@command -v sha256sum >/dev/null 2>&1 || { echo "❌ 未找到 sha256sum 命令"; exit 1; }
	@command -v find >/dev/null 2>&1 || { echo "❌ 未找到 find 命令"; exit 1; }
	@set -eu; \
		[ -d "$(BUILD_ROOT)" ] || { echo "❌ 未找到 $(WEB_ENV) 环境发布包，请先执行 make $(WEB_ENV)"; exit 1; }; \
		ZIP_FILE=$$(find "$(BUILD_ROOT)" -maxdepth 1 -type f -name "$(PACKAGE_PREFIX).*.zip" -printf '%T@ %p\n' | sort -nr | head -n 1 | cut -d' ' -f2-); \
		[ -n "$$ZIP_FILE" ] || { echo "❌ 未找到 $(WEB_ENV) 环境发布包，请先执行 make $(WEB_ENV)"; exit 1; }; \
		ZIP_NAME=$$(basename "$$ZIP_FILE" .zip); \
		EXPECTED_SHA256=$${ZIP_NAME##*.}; \
		ACTUAL_SHA256=$$(sha256sum "$$ZIP_FILE" | awk '{print $$1}'); \
		[ "$$EXPECTED_SHA256" = "$$ACTUAL_SHA256" ] || { echo "❌ SHA-256 校验失败: $$ZIP_FILE"; exit 1; }; \
		unzip -t "$$ZIP_FILE" >/dev/null; \
		UNPACK_DIR="$(BUILD_ROOT)/$$ZIP_NAME"; \
		rm -rf "$$UNPACK_DIR"; \
		mkdir -p "$$UNPACK_DIR"; \
		unzip -oq "$$ZIP_FILE" -d "$$UNPACK_DIR"; \
		test -f "$$UNPACK_DIR/index.html" || { echo "❌ 解压产物缺少 index.html"; exit 1; }; \
		test -d "$$UNPACK_DIR/assets" || { echo "❌ 解压产物缺少 assets/"; exit 1; }; \
		echo "✅ 解压完成: $$UNPACK_DIR"
