package service

import (
	"context"
	"errors"
	"testing"
	"time"

	adminConfig "github.com/GoAdminGroup/go-admin/modules/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"my-base/app/models"
	"my-base/app/service/dto"
	base "my-base/code/service"
	"my-base/module/notification"
)

func init() {
	if adminConfig.GetAuthUserTable() == "" {
		adminConfig.Initialize(&adminConfig.Config{AuthUserTable: "plan_test_users", InfoLogOff: true, ErrorLogOff: true, AccessLogOff: true})
	}
}

// recordingSender 模拟发送结果，验证业务流程而不发送真实邮件。
type recordingSender struct {
	calls int
	err   error
}

func (s *recordingSender) Send(_ context.Context, _ notification.Message) error {
	s.calls++
	return s.err
}

// newPlanTestDB 创建独立数据库和后台用户。
func newPlanTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(&models.SystemPlan{}, &models.SystemNotification{}); err != nil {
		t.Fatal(err)
	}
	table := adminConfig.GetAuthUserTable()
	if err = db.Table(table).AutoMigrate(&struct {
		ID int64 `gorm:"primaryKey"`
	}{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{1, 2} {
		if err = db.Table(table).Create(map[string]any{"id": id}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

// TestPlanLifecycle 验证权限、修改、到期幂等、发送和完成相互独立。
func TestPlanLifecycle(t *testing.T) {
	db := newPlanTestDB(t)
	s := SystemPlan{Service: base.Service{Orm: db}}
	in := dto.SystemPlanInput{UserID: 1, AssigneeID: 2, IsSuperAdmin: true, Title: "执行检查", Content: "请检查服务", Recipient: "user@qq.com", ExecuteAt: time.Now().Add(time.Hour)}
	if err := s.Save(&in); err != nil {
		t.Fatal(err)
	}
	var p models.SystemPlan
	db.First(&p)
	edit := in
	edit.ID = p.ID
	edit.UserID = 2
	edit.IsSuperAdmin = false
	if err := s.Save(&edit); err == nil {
		t.Fatal("执行人不应修改其他用户创建的计划")
	}
	if err := s.Transition(&dto.SystemPlanInput{ID: p.ID, UserID: 1, Action: "completed"}); err == nil {
		t.Fatal("创建人不能代执行人确认完成")
	}
	db.Model(&p).Update("execute_at", time.Now().UTC().Add(-time.Minute))
	for i := 0; i < 2; i++ {
		if err := s.GenerateDue(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	db.Model(&models.SystemNotification{}).Count(&count)
	if count != 1 {
		t.Fatalf("重复生成通知: %d", count)
	}
	sender := &recordingSender{}
	ns := SystemNotification{Service: base.Service{Orm: db}, Senders: map[string]notification.Sender{"email": sender}}
	for i := 0; i < 2; i++ {
		if err := ns.Dispatch(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if sender.calls != 1 {
		t.Fatalf("发送次数: %d", sender.calls)
	}
	db.First(&p, p.ID)
	if p.Status != "pending" {
		t.Fatal("发送成功不应完成计划")
	}
	if err := s.Transition(&dto.SystemPlanInput{ID: p.ID, UserID: 2, Action: "completed"}); err != nil {
		t.Fatal(err)
	}
}

// TestNotificationRetryAndRecovery 验证临时失败重试、领取恢复、永久失败和人工重试权限。
func TestNotificationRetryAndRecovery(t *testing.T) {
	db := newPlanTestDB(t)
	p := models.SystemPlan{CreatedBy: 1, AssigneeID: 2, Status: "pending", Channel: "email", Recipient: "user@example.com", Title: "测试", ExecuteAt: time.Now().UTC().Add(-time.Hour)}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	ps := SystemPlan{Service: base.Service{Orm: db}}
	if err := ps.GenerateDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	sender := &recordingSender{err: errors.New("临时故障")}
	ns := SystemNotification{Service: base.Service{Orm: db}, Senders: map[string]notification.Sender{"email": sender}}
	if err := ns.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	var n models.SystemNotification
	db.First(&n)
	if n.Status != "pending" || n.Attempts != 1 || !n.NextRetryAt.After(time.Now()) {
		t.Fatalf("重试状态错误: %+v", n)
	}
	if err := ns.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sender.calls != 1 {
		t.Fatal("重试时间前不应发送")
	}
	// 模拟进程在领取后退出，过期记录应恢复。
	db.Model(&n).Updates(map[string]any{"status": "sending", "locked_until": time.Now().UTC().Add(-time.Minute), "claim_token": "old"})
	sender.err = &notification.DeliveryError{Summary: "拒绝接收", Permanent: true}
	if err := ns.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	db.First(&n, n.ID)
	if n.Status != "failed" || n.Attempts != 2 {
		t.Fatalf("永久失败状态错误: %+v", n)
	}
	if err := ns.Retry(&dto.SystemNotificationRetryInput{ID: n.ID, UserID: 2}); err == nil {
		t.Fatal("接收人无权重试")
	}
	if err := ns.Retry(&dto.SystemNotificationRetryInput{ID: n.ID, UserID: 1}); err != nil {
		t.Fatal(err)
	}
	sender.err = nil
	if err := ns.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	db.First(&n, n.ID)
	if n.Status != "sent" || n.Attempts != 1 {
		t.Fatalf("人工重试失败: %+v", n)
	}
}

// TestCancelledPlanStopsNotification 验证取消和完成都停止未发送通知。
func TestCancelledPlanStopsNotification(t *testing.T) {
	for _, action := range []string{"cancelled", "completed"} {
		t.Run(action, func(t *testing.T) {
			db := newPlanTestDB(t)
			p := models.SystemPlan{CreatedBy: 1, AssigneeID: 1, Status: "pending", Channel: "email", Recipient: "user@example.com", ExecuteAt: time.Now().UTC().Add(-time.Hour)}
			db.Create(&p)
			ps := SystemPlan{Service: base.Service{Orm: db}}
			if err := ps.GenerateDue(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := ps.Transition(&dto.SystemPlanInput{ID: p.ID, UserID: 1, Action: action}); err != nil {
				t.Fatal(err)
			}
			sender := &recordingSender{}
			ns := SystemNotification{Service: base.Service{Orm: db}, Senders: map[string]notification.Sender{"email": sender}}
			if err := ns.Dispatch(context.Background()); err != nil {
				t.Fatal(err)
			}
			if sender.calls != 0 {
				t.Fatal("结束的计划不应发送")
			}
			var n models.SystemNotification
			db.First(&n)
			if n.Status != "cancelled" {
				t.Fatal(n.Status)
			}
		})
	}
}

// senderFunc 支持在模拟发送期间检查并发领取和领取令牌更新。
type senderFunc func(context.Context, notification.Message) error

func (f senderFunc) Send(ctx context.Context, m notification.Message) error { return f(ctx, m) }

// TestNotificationClaimOwnership 验证并发任务不会重发且旧领取者不能覆盖新结果。
func TestNotificationClaimOwnership(t *testing.T) {
	db := newPlanTestDB(t)
	p := models.SystemPlan{CreatedBy: 1, AssigneeID: 1, Status: "pending", Channel: "email", Recipient: "user@example.com", ExecuteAt: time.Now().UTC().Add(-time.Hour)}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	ps := SystemPlan{Service: base.Service{Orm: db}}
	if err := ps.GenerateDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	ns := SystemNotification{Service: base.Service{Orm: db}, Senders: map[string]notification.Sender{"email": senderFunc(func(_ context.Context, _ notification.Message) error {
		close(entered)
		<-release
		return errors.New("旧发送失败")
	})}}
	result := make(chan error, 1)
	go func() { result <- ns.Dispatch(context.Background()) }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("发送未开始")
	}
	if err := ns.Dispatch(context.Background()); err != nil {
		t.Fatal(err)
	}
	// 模拟领取过期后另一个任务已经成功，旧任务不得写回失败状态。
	if err := db.Model(&models.SystemNotification{}).Where("system_plan_id = ?", p.ID).Updates(map[string]any{"status": "sent", "claim_token": "new-owner"}).Error; err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	var n models.SystemNotification
	if err := db.First(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n.Status != "sent" || n.ClaimToken != "new-owner" {
		t.Fatalf("旧领取者覆盖了新结果: %+v", n)
	}
}
