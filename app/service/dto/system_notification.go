package dto

// SystemNotificationRetryInput 携带人工重试所需的记录标识和可信身份。
type SystemNotificationRetryInput struct {
	ID           uint
	UserID       int64
	IsSuperAdmin bool
}
