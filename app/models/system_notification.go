package models

import (
	"gorm.io/gorm"
	"time"
)

func init() {
	ModelManage.SetModel("system_notifications", &SystemNotification{}, "通知发送记录")
}

// SystemNotification 保存消息快照及可恢复的发送状态。
type SystemNotification struct {
	gorm.Model
	SystemPlanID    uint      `gorm:"not null;uniqueIndex:idx_plan_channel"`
	RecipientUserID int64     `gorm:"not null;index"`
	Channel         string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_plan_channel"`
	Recipient       string    `gorm:"type:varchar(254);not null"`
	Subject         string    `gorm:"type:varchar(200);not null"`
	Content         string    `gorm:"type:text;not null"`
	Status          string    `gorm:"type:varchar(20);not null;index:idx_notification_due,priority:1"`
	Attempts        int       `gorm:"not null;default:0"`
	NextRetryAt     time.Time `gorm:"not null;index:idx_notification_due,priority:2"`
	ClaimToken      string    `gorm:"type:varchar(64)"`
	LockedUntil     *time.Time
	SentAt          *time.Time
	LastError       string `gorm:"type:text"`
}

// TableName 返回通知记录表名。
func (SystemNotification) TableName() string { return "system_notifications" }
