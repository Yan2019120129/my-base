package apis

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"my-base/app/models"
	"my-base/app/service"
	"my-base/app/service/dto"
	"my-base/code/api"

	adminModels "github.com/GoAdminGroup/go-admin/plugins/admin/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SystemFileShare 提供系统文件分享接口。
type SystemFileShare struct {
	api.Api
}

// ShareList 返回当前后台用户可管理文件的分享记录。
func (e SystemFileShare) ShareList(ctx *gin.Context) {
	s := service.SystemFileShare{}
	if err := e.MakeContext(ctx).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(http.StatusInternalServerError, err, err.Error())
		return
	}

	fileID, ok := e.parseFileID(ctx)
	if !ok {
		return
	}
	user, ok := e.shareUser(ctx)
	if !ok {
		return
	}
	item := models.SystemFile{}
	if err := s.GetManageableFile(&dto.SystemFileShareManageFileInput{
		FileID:       fileID,
		IsSuperAdmin: user.IsSuperAdmin(),
	}, &item); err != nil {
		e.handleShareFileError(err)
		return
	}

	shares := make([]models.SystemFileShare, 0)
	if err := s.List(&dto.SystemFileShareListInput{FileID: fileID}, &shares); err != nil {
		e.Error(http.StatusInternalServerError, err, err.Error())
		return
	}
	e.OK(e.shareItems(ctx, shares), "查询成功")
}

// CreateShare 创建并返回可公开下载的文件分享链接。
func (e SystemFileShare) CreateShare(ctx *gin.Context) {
	s := service.SystemFileShare{}
	if err := e.MakeContext(ctx).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(http.StatusInternalServerError, err, err.Error())
		return
	}

	fileID, ok := e.parseFileID(ctx)
	if !ok {
		return
	}
	user, ok := e.shareUser(ctx)
	if !ok {
		return
	}
	payload := dto.SystemFileShareCreatePayload{}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		e.Error(http.StatusBadRequest, err, "invalid json body")
		return
	}

	item := models.SystemFile{}
	share, err := s.Create(&dto.SystemFileShareCreateInput{
		FileID:        fileID,
		DurationHours: payload.DurationHours,
		CreatedBy:     user.Id,
		IsSuperAdmin:  user.IsSuperAdmin(),
	}, &item)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			e.Error(http.StatusNotFound, err, "file not found")
			return
		}
		if errors.Is(err, service.ErrInvalidShareDuration) {
			e.Error(http.StatusBadRequest, err, err.Error())
			return
		}
		e.Error(http.StatusInternalServerError, err, err.Error())
		return
	}
	result := e.shareItem(ctx, *share)
	result.OriginalName = item.OriginalName
	e.OK(result, "创建成功")
}

// RevokeShare 撤销指定文件的分享链接。
func (e SystemFileShare) RevokeShare(ctx *gin.Context) {
	s := service.SystemFileShare{}
	if err := e.MakeContext(ctx).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(http.StatusInternalServerError, err, err.Error())
		return
	}

	fileID, ok := e.parseFileID(ctx)
	if !ok {
		return
	}
	user, ok := e.shareUser(ctx)
	if !ok {
		return
	}
	item := models.SystemFile{}
	if err := s.GetManageableFile(&dto.SystemFileShareManageFileInput{
		FileID:       fileID,
		IsSuperAdmin: user.IsSuperAdmin(),
	}, &item); err != nil {
		e.handleShareFileError(err)
		return
	}
	shareID, err := strconv.ParseUint(ctx.Param("shareId"), 10, strconv.IntSize)
	if err != nil || shareID <= 0 {
		e.Error(http.StatusBadRequest, errors.New("invalid share id"), "invalid share id")
		return
	}
	if err := s.Revoke(&dto.SystemFileShareRevokeInput{FileID: fileID, ShareID: uint(shareID)}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			e.Error(http.StatusNotFound, err, "share not found")
			return
		}
		e.Error(http.StatusInternalServerError, err, err.Error())
		return
	}
	e.OK(nil, "撤销成功")
}

// ShareDownload 根据分享令牌下载文件，无需后台登录。
func (e SystemFileShare) ShareDownload(ctx *gin.Context) {
	s := service.SystemFileShare{}
	if err := e.MakeContext(ctx).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(http.StatusInternalServerError, err, err.Error())
		return
	}

	item := models.SystemFile{}
	if err := s.GetDownloadFile(&dto.SystemFileShareDownloadInput{Token: strings.TrimSpace(ctx.Param("token"))}, &item); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			e.Error(http.StatusNotFound, err, "file not found")
			return
		}
		e.Error(http.StatusInternalServerError, err, err.Error())
		return
	}
	SystemFile{Api: e.Api}.serveDownload(ctx, &item)
}

// parseFileID 解析请求路径中的文件标识。
func (e SystemFileShare) parseFileID(ctx *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, strconv.IntSize)
	if err != nil || id <= 0 {
		e.Error(http.StatusBadRequest, errors.New("invalid id"), "invalid id")
		return 0, false
	}
	return uint(id), true
}

// shareUser 获取已通过文件管理权限校验的后台用户。
func (e SystemFileShare) shareUser(ctx *gin.Context) (adminModels.UserModel, bool) {
	user, ok := ctx.Get("user")
	if !ok {
		e.Error(http.StatusInternalServerError, errors.New("admin user not found"), "admin user not found")
		return adminModels.UserModel{}, false
	}
	result, ok := user.(adminModels.UserModel)
	if !ok {
		e.Error(http.StatusInternalServerError, errors.New("invalid admin user"), "invalid admin user")
		return adminModels.UserModel{}, false
	}
	return result, true
}

// handleShareFileError 返回文件不可管理时的统一响应。
func (e SystemFileShare) handleShareFileError(err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		e.Error(http.StatusNotFound, err, "file not found")
		return
	}
	e.Error(http.StatusInternalServerError, err, err.Error())
}

// shareItems 将分享记录转换为接口返回内容。
func (e SystemFileShare) shareItems(ctx *gin.Context, shares []models.SystemFileShare) []dto.SystemFileShareItem {
	items := make([]dto.SystemFileShareItem, 0, len(shares))
	for _, share := range shares {
		items = append(items, e.shareItem(ctx, share))
	}
	return items
}

// shareItem 将单条分享记录转换为接口返回内容。
func (e SystemFileShare) shareItem(ctx *gin.Context, share models.SystemFileShare) dto.SystemFileShareItem {
	return dto.SystemFileShareItem{
		Id:          share.ID,
		DownloadURL: shareDownloadURL(ctx, share.Token),
		ExpiresAt:   share.ExpiresAt,
		RevokedAt:   share.RevokedAt,
		CreatedAt:   share.CreatedAt,
		Status:      shareStatus(share),
	}
}

// shareStatus 计算分享记录当前状态。
func shareStatus(share models.SystemFileShare) string {
	if share.RevokedAt != nil {
		return "已撤销"
	}
	if !share.ExpiresAt.After(time.Now()) {
		return "已过期"
	}
	return "有效"
}

// shareDownloadURL 生成当前请求站点下的完整分享下载链接。
func shareDownloadURL(ctx *gin.Context, token string) string {
	scheme := "http"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(ctx.GetHeader("X-Forwarded-Proto")); forwarded != "" {
		candidate := strings.ToLower(strings.TrimSpace(strings.Split(forwarded, ",")[0]))
		if candidate == "http" || candidate == "https" {
			scheme = candidate
		}
	}
	return (&url.URL{Scheme: scheme, Host: ctx.Request.Host, Path: "/shares/" + token + "/download"}).String()
}
