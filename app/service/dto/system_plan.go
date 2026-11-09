package dto

import "time"

// SystemPlanInput 携带可信会话身份及计划表单参数。
type SystemPlanInput struct {
	ID           uint
	UserID       int64
	IsSuperAdmin bool
	Title        string
	Content      string
	AssigneeID   int64
	Recipient    string
	ExecuteAt    time.Time
	Action       string
}
