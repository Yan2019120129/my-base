package dto

import "time"

// SystemFileShareCreatePayload 定义创建文件分享链接的 HTTP 请求体。
type SystemFileShareCreatePayload struct {
	DurationHours int `json:"durationHours"`
}

// SystemFileShareManageFileInput 定义查询当前用户可管理系统文件的服务输入。
type SystemFileShareManageFileInput struct {
	FileID       uint
	IsSuperAdmin bool
}

// SystemFileShareCreateInput 定义创建系统文件分享记录的服务输入。
type SystemFileShareCreateInput struct {
	FileID        uint
	DurationHours int
	CreatedBy     int64
	IsSuperAdmin  bool
}

// SystemFileShareListInput 定义查询系统文件分享记录的服务输入。
type SystemFileShareListInput struct {
	FileID uint
}

// SystemFileShareRevokeInput 定义撤销系统文件分享记录的服务输入。
type SystemFileShareRevokeInput struct {
	FileID  uint
	ShareID uint
}

// SystemFileShareUpdateExpiresAtInput 定义修改分享链接到期时间的服务输入。
type SystemFileShareUpdateExpiresAtInput struct {
	ShareID      uint
	ExpiresAt    time.Time
	IsSuperAdmin bool
}

// SystemFileShareRevokeByIDInput 定义按分享记录标识撤销的服务输入。
type SystemFileShareRevokeByIDInput struct {
	ShareID      uint
	IsSuperAdmin bool
}

// SystemFileShareDownloadInput 定义根据令牌下载分享文件的服务输入。
type SystemFileShareDownloadInput struct {
	Token string
}

// SystemFileShareItem 定义文件分享记录的接口返回内容。
type SystemFileShareItem struct {
	OriginalName string     `json:"originalName,omitempty"`
	Id           uint       `json:"id"`
	DownloadURL  string     `json:"downloadUrl"`
	ExpiresAt    time.Time  `json:"expiresAt"`
	RevokedAt    *time.Time `json:"revokedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	Status       string     `json:"status"`
}
