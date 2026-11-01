package html

import (
	"fmt"
	"html/template"
	"strings"
)

// SystemFile 提供系统文件表格使用的 HTML、JavaScript 和 CSS 对象。
type SystemFile struct{}

// GetSystemFile 获取无状态的系统文件 HTML 对象。
func GetSystemFile() *SystemFile {
	return &SystemFile{}
}

// Status 根据状态值返回系统文件状态标签。
func (s *SystemFile) Status(value string) template.HTML {
	switch value {
	case "1":
		return template.HTML(`<span class="label label-success">正常</span>`)
	case "0":
		return template.HTML(`<span class="label label-warning">禁用</span>`)
	default:
		return template.HTML(`<span class="label label-default">删除</span>`)
	}
}

func (s *SystemFile) Preview(publicURL string, mimeType interface{}) template.HTML {
	url := template.HTMLEscapeString(publicURL)
	if url == "" {
		return template.HTML(`<span class="text-muted">无文件</span>`)
	}
	mime := fmt.Sprint(mimeType)
	if strings.HasPrefix(mime, "image/") {
		return template.HTML(`<a href="` + url + `" target="_blank"><img src="` + url + `" style="max-width:80px;max-height:80px;object-fit:contain;border:1px solid #eee" /></a>`)
	}
	return template.HTML(`<a href="` + url + `" target="_blank">打开</a>`)
}

func (s *SystemFile) DownloadButton(id string) template.HTML {
	id = template.HTMLEscapeString(id)
	return template.HTML(`<a class="btn btn-xs btn-primary" href="/admin/files/download/` + id + `" target="_blank">下载</a>`)
}

// QuickShareLink 渲染创建一天有效期分享链接的 URL 入口。
func (s *SystemFile) QuickShareLink(id, publicURL string) template.HTML {
	id = template.HTMLEscapeString(id)
	publicURL = template.HTMLEscapeString(publicURL)
	if publicURL == "" {
		publicURL = "创建 1 天分享链接"
	}
	return template.HTML(`<button type="button" class="btn btn-link btn-xs system-file-quick-share-link" data-file-id="` + id + `">` + publicURL + `</button>`)
}

// ShareButton 渲染打开文件分享弹窗的按钮。
func (s *SystemFile) ShareButton(id string) template.HTML {
	id = template.HTMLEscapeString(id)
	return template.HTML(`<button type="button" class="btn btn-xs btn-success system-file-share-button" data-file-id="` + id + `">分享</button>`)
}

// ShareModal 渲染分享时长选择弹窗及交互脚本。
func (s *SystemFile) ShareModal() template.HTML {
	return template.HTML(`
<style>
/* =========================
   文件分享弹窗
   ========================= */

#system-file-share-modal .modal-dialog {
	width: auto;
	max-width: 560px;
	margin: 60px auto;
	padding: 0 16px;
}

#system-file-share-modal .modal-content {
	overflow: hidden;
	border: 0;
	border-radius: 14px;
	background: #ffffff;
	box-shadow:
		0 24px 60px rgba(15, 23, 42, 0.18),
		0 8px 20px rgba(15, 23, 42, 0.08);
}

/* 头部 */
#system-file-share-modal .modal-header {
	position: relative;
	display: flex;
	align-items: center;
	justify-content: space-between;
	min-height: 76px;
	padding: 20px 24px;
	border-bottom: 1px solid #edf0f5;
	background:
		linear-gradient(
			135deg,
			rgba(59, 130, 246, 0.10),
			rgba(99, 102, 241, 0.06)
		),
		#ffffff;
}

#system-file-share-modal .system-file-share-title-wrap {
	display: flex;
	align-items: center;
	min-width: 0;
}

#system-file-share-modal .system-file-share-title-icon {
	display: flex;
	flex: 0 0 42px;
	align-items: center;
	justify-content: center;
	width: 42px;
	height: 42px;
	margin-right: 13px;
	border-radius: 12px;
	background: linear-gradient(135deg, #3b82f6, #6366f1);
	color: #ffffff;
	font-size: 22px;
	font-weight: 600;
	box-shadow: 0 8px 18px rgba(59, 130, 246, 0.24);
}

#system-file-share-modal .modal-title {
	margin: 0;
	color: #172033;
	font-size: 18px;
	font-weight: 600;
	line-height: 1.4;
}

#system-file-share-modal .system-file-share-subtitle {
	margin-top: 3px;
	color: #8992a3;
	font-size: 12px;
	line-height: 1.5;
}

#system-file-share-modal .close {
	position: static;
	display: flex;
	align-items: center;
	justify-content: center;
	width: 34px;
	height: 34px;
	margin: 0;
	padding: 0;
	border: 0;
	border-radius: 9px;
	background: transparent;
	color: #8a94a6;
	font-size: 25px;
	font-weight: 400;
	line-height: 1;
	opacity: 1;
	text-shadow: none;
	transition:
		background-color 0.2s ease,
		color 0.2s ease,
		transform 0.2s ease;
}

#system-file-share-modal .close:hover,
#system-file-share-modal .close:focus {
	background: #eef2f7;
	color: #334155;
	opacity: 1;
	outline: none;
	transform: rotate(4deg);
}

/* 内容区 */
#system-file-share-modal .modal-body {
	padding: 24px;
}

#system-file-share-modal .system-file-share-description {
	display: flex;
	align-items: flex-start;
	margin-bottom: 20px;
	padding: 13px 15px;
	border: 1px solid #e6edf7;
	border-radius: 10px;
	background: #f8faff;
	color: #64748b;
	font-size: 13px;
	line-height: 1.7;
}

#system-file-share-modal .system-file-share-description-icon {
	flex: 0 0 auto;
	margin-right: 9px;
	color: #3b82f6;
	font-size: 16px;
	line-height: 1.5;
}

#system-file-share-modal .system-file-share-section-title {
	display: block;
	margin-bottom: 10px;
	color: #344054;
	font-size: 13px;
	font-weight: 600;
}

/* 有效期按钮 */
#system-file-share-modal .system-file-share-duration-buttons {
	display: grid;
	grid-template-columns: repeat(3, minmax(0, 1fr));
	gap: 12px;
}

#system-file-share-modal .system-file-share-create {
	position: relative;
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	justify-content: center;
	min-height: 78px;
	padding: 14px 15px;
	overflow: hidden;
	border: 1px solid #dce5f2;
	border-radius: 11px;
	background: #ffffff;
	color: #344054;
	text-align: left;
	box-shadow: 0 3px 10px rgba(15, 23, 42, 0.04);
	transition:
		border-color 0.2s ease,
		background-color 0.2s ease,
		box-shadow 0.2s ease,
		color 0.2s ease,
		transform 0.2s ease;
}

#system-file-share-modal .system-file-share-create::after {
	position: absolute;
	right: -14px;
	bottom: -18px;
	width: 54px;
	height: 54px;
	border-radius: 50%;
	background: rgba(59, 130, 246, 0.07);
	content: "";
	transition: transform 0.2s ease;
}

#system-file-share-modal .system-file-share-create:hover,
#system-file-share-modal .system-file-share-create:focus {
	border-color: #6ea8fe;
	background: #f8fbff;
	color: #2563eb;
	box-shadow: 0 9px 22px rgba(59, 130, 246, 0.13);
	outline: none;
	transform: translateY(-2px);
}

#system-file-share-modal .system-file-share-create:hover::after {
	transform: scale(1.25);
}

#system-file-share-modal .system-file-share-create:active {
	transform: translateY(0);
}

#system-file-share-modal .system-file-share-create[disabled] {
	border-color: #e5e7eb;
	background: #f8fafc;
	color: #94a3b8;
	box-shadow: none;
	cursor: not-allowed;
	opacity: 0.75;
	transform: none;
}

#system-file-share-modal .system-file-share-duration {
	position: relative;
	z-index: 1;
	font-size: 16px;
	font-weight: 600;
	line-height: 1.4;
}

#system-file-share-modal .system-file-share-duration-tip {
	position: relative;
	z-index: 1;
	margin-top: 4px;
	color: #98a2b3;
	font-size: 11px;
	font-weight: 400;
	line-height: 1.4;
}

/* 分享结果 */
#system-file-share-modal .system-file-share-result {
	margin-top: 22px;
	padding-top: 20px;
	border-top: 1px solid #edf0f5;
}

#system-file-share-modal .system-file-share-result-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-bottom: 9px;
}

#system-file-share-modal .system-file-share-result label {
	margin: 0;
	color: #344054;
	font-size: 13px;
	font-weight: 600;
}

#system-file-share-modal .system-file-share-result-state {
	display: inline-flex;
	align-items: center;
	color: #16a34a;
	font-size: 12px;
}

#system-file-share-modal .system-file-share-result-state::before {
	display: inline-block;
	width: 7px;
	height: 7px;
	margin-right: 6px;
	border-radius: 50%;
	background: #22c55e;
	box-shadow: 0 0 0 3px rgba(34, 197, 94, 0.13);
	content: "";
}

#system-file-share-modal .system-file-share-link-group {
	display: flex;
	align-items: stretch;
	width: 100%;
	padding: 4px;
	border: 1px solid #dce4ef;
	border-radius: 10px;
	background: #f8fafc;
	transition:
		border-color 0.2s ease,
		box-shadow 0.2s ease;
}

#system-file-share-modal .system-file-share-link-group:focus-within {
	border-color: #77a9f8;
	box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.10);
}

#system-file-share-modal .system-file-share-link {
	flex: 1;
	width: 1%;
	height: 38px;
	padding: 8px 10px;
	border: 0;
	background: transparent;
	color: #475467;
	font-size: 13px;
	box-shadow: none;
}

#system-file-share-modal .system-file-share-link:focus {
	border: 0;
	outline: none;
	box-shadow: none;
}

#system-file-share-modal .system-file-share-copy {
	flex: 0 0 auto;
	height: 38px;
	padding: 0 16px;
	border: 0;
	border-radius: 8px;
	background: #2563eb;
	color: #ffffff;
	font-size: 13px;
	font-weight: 500;
	box-shadow: 0 4px 10px rgba(37, 99, 235, 0.20);
	transition:
		background-color 0.2s ease,
		box-shadow 0.2s ease,
		transform 0.2s ease;
}

#system-file-share-modal .system-file-share-copy:hover,
#system-file-share-modal .system-file-share-copy:focus {
	background: #1d4ed8;
	color: #ffffff;
	outline: none;
	box-shadow: 0 6px 14px rgba(37, 99, 235, 0.25);
}

#system-file-share-modal .system-file-share-copy:active {
	transform: scale(0.98);
}

/* 成功提示 */
#system-file-share-modal .system-file-share-notice {
	margin-top: 12px;
	margin-bottom: 0;
	padding: 10px 13px;
	border: 1px solid #bbf7d0;
	border-radius: 9px;
	background: #f0fdf4;
	color: #15803d;
	font-size: 13px;
	line-height: 1.5;
}

/* 移动端 */
@media (max-width: 767px) {
	#system-file-share-modal .modal-dialog {
		margin: 20px auto;
		padding: 0 12px;
	}

	#system-file-share-modal .modal-header {
		min-height: 68px;
		padding: 16px 18px;
	}

	#system-file-share-modal .modal-body {
		padding: 18px;
	}

	#system-file-share-modal .system-file-share-duration-buttons {
		grid-template-columns: 1fr;
		gap: 9px;
	}

	#system-file-share-modal .system-file-share-create {
		min-height: 64px;
	}

	#system-file-share-modal .system-file-share-link-group {
		display: block;
	}

	#system-file-share-modal .system-file-share-link {
		display: block;
		width: 100%;
	}

	#system-file-share-modal .system-file-share-copy {
		display: block;
		width: 100%;
		margin-top: 4px;
	}
}
</style>

<div
  class="modal fade"
  id="system-file-share-modal"
  tabindex="-1"
  role="dialog"
  aria-labelledby="system-file-share-modal-title"
  aria-hidden="true"
>
  <div class="modal-dialog" role="document">
    <div class="modal-content">

      <div class="modal-header">
        <div class="system-file-share-title-wrap">
          <div class="system-file-share-title-icon" aria-hidden="true">↗</div>

          <div>
            <h4 class="modal-title" id="system-file-share-modal-title">
              文件分享
            </h4>
            <div class="system-file-share-subtitle">
              创建一个具有有效期的公开下载链接
            </div>
          </div>
        </div>

        <button
          type="button"
          class="close"
          data-dismiss="modal"
          aria-label="关闭"
        >
          <span aria-hidden="true">&times;</span>
        </button>
      </div>

      <div class="modal-body">
        <div class="system-file-share-description">
          <span class="system-file-share-description-icon" aria-hidden="true">ⓘ</span>
          <span>
            请选择分享链接的有效期。每次操作都会创建一个新的公开下载链接，
            请谨慎分享给可信用户。
          </span>
        </div>

        <span class="system-file-share-section-title">
          选择有效期
        </span>

        <div
          class="system-file-share-duration-buttons"
          role="group"
          aria-label="分享链接有效期"
        >
          <button
            type="button"
            class="btn btn-primary system-file-share-create"
            data-hours="24"
          >
            <span class="system-file-share-duration">1 天</span>
            <span class="system-file-share-duration-tip">适合临时分享</span>
          </button>

          <button
            type="button"
            class="btn btn-primary system-file-share-create"
            data-hours="168"
          >
            <span class="system-file-share-duration">7 天</span>
            <span class="system-file-share-duration-tip">适合短期协作</span>
          </button>

          <button
            type="button"
            class="btn btn-primary system-file-share-create"
            data-hours="720"
          >
            <span class="system-file-share-duration">30 天</span>
            <span class="system-file-share-duration-tip">适合长期访问</span>
          </button>
        </div>

        <div class="system-file-share-result" style="display:none">
          <div class="system-file-share-result-header">
            <label for="system-file-share-link">分享链接</label>
            <span class="system-file-share-result-state">创建成功</span>
          </div>

          <div class="system-file-share-link-group">
            <input
              id="system-file-share-link"
              type="text"
              class="form-control system-file-share-link"
              readonly
            >

            <button
              type="button"
              class="btn btn-default system-file-share-copy"
            >
              复制链接
            </button>
          </div>
        </div>

        <div
          class="alert alert-success system-file-share-notice"
          role="status"
          style="display:none"
        ></div>
      </div>

    </div>
  </div>
</div>

<script>
(function () {
  var modal = $('#system-file-share-modal');
  var noticeTimer;

  function request(path, method, body) {
    return $.ajax({
      url: path,
      type: method,
      contentType: 'application/json',
      dataType: 'json',
      data: body ? JSON.stringify(body) : undefined
    });
  }

  function errorMessage(response) {
    if (response && response.responseJSON && response.responseJSON.msg) {
      return response.responseJSON.msg;
    }
    return '请求失败，请稍后重试';
  }

  function copyLink(link) {
    if (navigator.clipboard && window.isSecureContext) {
      return navigator.clipboard.writeText(link);
    }

    var input = document.createElement('textarea');
    input.value = link;
    input.style.position = 'fixed';
    input.style.opacity = '0';

    document.body.appendChild(input);
    input.select();

    var copied = document.execCommand('copy');

    document.body.removeChild(input);

    return copied
      ? Promise.resolve()
      : Promise.reject(new Error('copy failed'));
  }

  function resetShareResult() {
    modal.find('.system-file-share-link').val('');
    modal.find('.system-file-share-result').hide();
  }

  function showShareResult(link) {
    modal.find('.system-file-share-link').val(link);
    modal.find('.system-file-share-result').show();
  }

  function showNotice(message) {
    var notice = modal.find('.system-file-share-notice');

    window.clearTimeout(noticeTimer);

    notice
      .text(message)
      .stop(true, true)
      .fadeIn(150);

    noticeTimer = window.setTimeout(function () {
      notice.fadeOut(300);
    }, 2000);
  }

  function createShare(fileID, durationHours, button) {
    if (!fileID) {
      alert('文件标识无效');
      return;
    }

    button.prop('disabled', true);

    request(
      '/admin/files/' + fileID + '/shares',
      'POST',
      {
        durationHours: durationHours
      }
    ).done(function (response) {
      if (
        !response ||
        response.status === 'error' ||
        !response.data ||
        !response.data.downloadUrl
      ) {
        alert(
          response && response.msg
            ? response.msg
            : '创建分享链接失败'
        );
        return;
      }

      showShareResult(response.data.downloadUrl);
    }).fail(function (response) {
      alert(errorMessage(response));
    }).always(function () {
      button.prop('disabled', false);
    });
  }

  $(document)
    .off('click.systemFileShare', '.system-file-share-button')
    .on(
      'click.systemFileShare',
      '.system-file-share-button',
      function () {
        modal.data('file-id', $(this).data('file-id'));
        resetShareResult();
        modal.modal('show');
      }
    );

  $(document)
    .off(
      'click.systemFileShareCreate',
      '.system-file-share-create'
    )
    .on(
      'click.systemFileShareCreate',
      '.system-file-share-create',
      function () {
        var button = $(this);

        createShare(
          modal.data('file-id'),
          Number(button.data('hours')),
          button
        );
      }
    );

  $(document)
    .off(
      'click.systemFileShareCopy',
      '.system-file-share-copy'
    )
    .on(
      'click.systemFileShareCopy',
      '.system-file-share-copy',
      function () {
        var link = modal.find('.system-file-share-link').val();

        if (!link) {
          return;
        }

        copyLink(link).then(function () {
          showNotice('分享链接已复制');
        }).catch(function () {
          window.prompt('请复制分享链接', link);
        });
      }
    );

  $(document)
    .off(
      'click.systemFileQuickShare',
      '.system-file-quick-share-link'
    )
    .on(
      'click.systemFileQuickShare',
      '.system-file-quick-share-link',
      function () {
        var button = $(this);

        modal.data('file-id', button.data('file-id'));
        resetShareResult();
        modal.modal('show');

        createShare(
          button.data('file-id'),
          24,
          button
        );
      }
    );
}());
</script>`)
}
