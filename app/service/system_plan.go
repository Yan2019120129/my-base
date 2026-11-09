package service

import (
	"context"
	"errors"
	"my-base/configs"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	adminConfig "github.com/GoAdminGroup/go-admin/modules/config"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"my-base/app/models"
	"my-base/app/service/dto"
	base "my-base/code/service"
)

// SystemPlan 管理计划状态与到期通知的事务一致性。
type SystemPlan struct{ base.Service }

// Save 创建或修改尚未到期的计划，身份只能来自后台会话。
func (s *SystemPlan) Save(in *dto.SystemPlanInput) error {
	if in == nil || in.UserID <= 0 {
		return errors.New("用户身份无效")
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Recipient = strings.TrimSpace(in.Recipient)
	if in.Title == "" || utf8.RuneCountInString(in.Title) > 200 || strings.ContainsAny(in.Title+in.Recipient, "\r\n") {
		return errors.New("标题或邮箱无效")
	}
	address, err := mail.ParseAddress(in.Recipient)
	if err != nil || address.Address != in.Recipient || len(in.Recipient) > 254 {
		return errors.New("请输入有效邮箱地址")
	}
	parts := strings.Split(in.Recipient, "@")
	allowed := false
	for _, domain := range configs.GetAllowedRecipientDomains() {
		if strings.EqualFold(parts[len(parts)-1], strings.TrimSpace(domain)) {
			allowed = true
			break
		}
	}
	if !allowed {
		return errors.New("不支持该邮箱后缀，目前支持：" + strings.Join(configs.GetAllowedRecipientDomains(), "、"))
	}
	in.Recipient = parts[0] + "@" + strings.ToLower(parts[len(parts)-1])
	if !in.ExecuteAt.After(time.Now()) {
		return errors.New("执行时间必须晚于当前时间")
	}
	if !in.IsSuperAdmin {
		in.AssigneeID = in.UserID
	}
	if in.AssigneeID <= 0 {
		return errors.New("执行用户无效")
	}
	return s.Orm.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table(adminConfig.GetAuthUserTable()).Where("id = ?", in.AssigneeID).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return errors.New("执行用户不存在")
		}
		if in.ID == 0 {
			return tx.Create(&models.SystemPlan{Title: in.Title, Content: in.Content, CreatedBy: in.UserID, AssigneeID: in.AssigneeID, Channel: "email", Recipient: in.Recipient, ExecuteAt: in.ExecuteAt.UTC(), Status: "pending"}).Error
		}
		var p models.SystemPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, in.ID).Error; err != nil {
			return err
		}
		if (!in.IsSuperAdmin && p.CreatedBy != in.UserID) || p.Status != "pending" || !p.ExecuteAt.After(time.Now()) {
			return errors.New("无权修改或计划已经到期")
		}
		return tx.Model(&p).Updates(map[string]any{"title": in.Title, "content": in.Content, "assignee_id": in.AssigneeID, "recipient": in.Recipient, "execute_at": in.ExecuteAt.UTC()}).Error
	})
}

// Transition 完成或取消计划，并停止尚未领取的通知。
func (s *SystemPlan) Transition(in *dto.SystemPlanInput) error {
	if in == nil || in.ID == 0 || in.UserID <= 0 {
		return errors.New("计划或用户无效")
	}
	if in.Action != "completed" && in.Action != "cancelled" {
		return errors.New("不支持的操作")
	}
	return s.Orm.Transaction(func(tx *gorm.DB) error {
		var p models.SystemPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, in.ID).Error; err != nil {
			return err
		}
		if p.Status != "pending" {
			return errors.New("计划不处于待执行状态")
		}
		if in.Action == "completed" && p.AssigneeID != in.UserID {
			return errors.New("仅执行用户可确认完成")
		}
		if in.Action == "cancelled" && !in.IsSuperAdmin && p.CreatedBy != in.UserID {
			return errors.New("无权取消计划")
		}
		now := time.Now().UTC()
		field := "completed_at"
		if in.Action == "cancelled" {
			field = "cancelled_at"
		}
		if err := tx.Model(&p).Updates(map[string]any{"status": in.Action, field: now}).Error; err != nil {
			return err
		}
		return tx.Model(&models.SystemNotification{}).Where("system_plan_id = ? AND status IN ?", p.ID, []string{"pending", "failed"}).Update("status", "cancelled").Error
	})
}

// GenerateDue 分批生成到期消息，唯一约束与事务避免重复生成。
func (s *SystemPlan) GenerateDue(ctx context.Context) error {
	var plans []models.SystemPlan
	now := time.Now().UTC()
	if err := s.Orm.WithContext(ctx).Where("status = ? AND execute_at <= ? AND notification_created_at IS NULL", "pending", now).Order("id").Limit(100).Find(&plans).Error; err != nil {
		return err
	}
	for _, candidate := range plans {
		if err := s.Orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var p models.SystemPlan
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, candidate.ID).Error; err != nil {
				return err
			}
			if p.Status != "pending" || p.NotificationCreatedAt != nil || p.ExecuteAt.After(now) {
				return nil
			}
			n := models.SystemNotification{SystemPlanID: p.ID, RecipientUserID: p.AssigneeID, Channel: p.Channel, Recipient: p.Recipient, Subject: p.Title, Content: p.Content, Status: "pending", NextRetryAt: now}
			if err := tx.Create(&n).Error; err != nil {
				return err
			}
			return tx.Model(&p).Update("notification_created_at", now).Error
		}); err != nil {
			return err
		}
	}
	return nil
}
