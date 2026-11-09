package models

import (
	"gorm.io/gorm"
	"time"
)

func init() { ModelManage.SetModel("system_plans", &SystemPlan{}, "执行计划") }

// SystemPlan 保存单次执行计划，完成状态独立于通知状态。
type SystemPlan struct {
	gorm.Model
	Title                 string    `gorm:"type:varchar(200);not null"`
	Content               string    `gorm:"type:text;not null"`
	CreatedBy             int64     `gorm:"not null;index"`
	AssigneeID            int64     `gorm:"not null;index"`
	Channel               string    `gorm:"type:varchar(32);not null"`
	Recipient             string    `gorm:"type:varchar(254);not null"`
	ExecuteAt             time.Time `gorm:"not null;index:idx_plan_due,priority:2"`
	Status                string    `gorm:"type:varchar(20);not null;index:idx_plan_due,priority:1"`
	CompletedAt           *time.Time
	CancelledAt           *time.Time
	NotificationCreatedAt *time.Time
}

// TableName 返回执行计划表名。
func (SystemPlan) TableName() string { return "system_plans" }
