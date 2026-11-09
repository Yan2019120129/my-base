package tasks

import (
	"context"
	"gorm.io/gorm"
	"my-base/app/service"
	base "my-base/code/service"
	"time"
)

// systemPlanJob 扫描到期计划并生成持久化通知。
func systemPlanJob(db *gorm.DB) Job {
	s := service.SystemPlan{Service: base.Service{Orm: db}}
	return Job{Name: "system_plan", Interval: 10 * time.Second, Run: func(ctx context.Context) error { return s.GenerateDue(ctx) }}
}
