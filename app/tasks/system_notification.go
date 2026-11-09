package tasks

import (
	"context"
	"gorm.io/gorm"
	"my-base/app/service"
	base "my-base/code/service"
	"my-base/configs"
	"my-base/module/notification"
	"os"
	"time"
)

// systemNotificationJob 在启动时组装渠道，关闭通知时不启动发送任务。
func systemNotificationJob(db *gorm.DB) (Job, error) {
	cfg := configs.GetConfig().Notification
	if cfg == nil || !cfg.Enabled {
		return Job{}, nil
	}
	password := cfg.Password
	if value, ok := os.LookupEnv("SMTP_PASSWORD"); ok {
		password = value
	}
	sender, err := notification.NewEmailSender(notification.EmailConfig{Host: cfg.Host, Port: cfg.Port, Username: cfg.Username, Password: password, From: cfg.From, TLSMode: cfg.TLSMode, Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second})
	if err != nil {
		return Job{}, err
	}
	s := service.SystemNotification{Service: base.Service{Orm: db}, Senders: map[string]notification.Sender{"email": sender}}
	return Job{Name: "system_notification", Interval: 5 * time.Second, Run: func(ctx context.Context) error { return s.Dispatch(ctx) }}, nil
}
