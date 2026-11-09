package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"my-base/app/models"
	"my-base/app/service/dto"
	base "my-base/code/service"
	"my-base/module/notification"
)

// SystemNotification 负责通知领取、发送和恢复，不包含渠道实现。
type SystemNotification struct {
	base.Service
	Senders map[string]notification.Sender
}

// Retry 仅允许有权管理计划的用户重试失败通知。
func (s *SystemNotification) Retry(in *dto.SystemNotificationRetryInput) error {
	if in == nil || in.ID == 0 || in.UserID <= 0 {
		return errors.New("记录或用户无效")
	}
	return s.Orm.Transaction(func(tx *gorm.DB) error {
		var n models.SystemNotification
		if err := tx.First(&n, in.ID).Error; err != nil {
			return err
		}
		var p models.SystemPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, n.SystemPlanID).Error; err != nil {
			return err
		}
		if p.Status != "pending" || (!in.IsSuperAdmin && p.CreatedBy != in.UserID) {
			return errors.New("无权重试或计划已经结束")
		}
		result := tx.Model(&models.SystemNotification{}).Where("id = ? AND status = ?", in.ID, "failed").Updates(map[string]any{"status": "pending", "attempts": 0, "next_retry_at": time.Now().UTC(), "last_error": "", "claim_token": "", "locked_until": nil})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("仅失败通知可重试")
		}
		return nil
	})
}

// Dispatch 分批发送，使用领取令牌防止过期任务覆盖新的发送结果。
func (s *SystemNotification) Dispatch(ctx context.Context) error {
	now := time.Now().UTC()
	var records []models.SystemNotification
	if err := s.Orm.WithContext(ctx).Where("(status = ? AND next_retry_at <= ?) OR (status = ? AND locked_until <= ?)", "pending", now, "sending", now).Order("id").Limit(20).Find(&records).Error; err != nil {
		return err
	}
	var failures []error
	for _, candidate := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		token := uuid.NewString()
		var n models.SystemNotification
		claimed := false
		err := s.Orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// 与计划取消及完成操作按相同顺序锁定计划行。
			var p models.SystemPlan
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, candidate.SystemPlanID).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&n, candidate.ID).Error; err != nil {
				return err
			}
			current := time.Now().UTC()
			query := tx.Model(&models.SystemNotification{}).Where("id = ? AND ((status = ? AND next_retry_at <= ?) OR (status = ? AND locked_until <= ?))", candidate.ID, "pending", current, "sending", current)
			if p.Status != "pending" {
				return query.Update("status", "cancelled").Error
			}
			if s.Senders[n.Channel] == nil {
				return nil
			}
			if n.Attempts >= 4 {
				return query.Updates(map[string]any{"status": "failed", "last_error": "发送中断且已达到尝试次数上限"}).Error
			}
			result := query.Updates(map[string]any{"status": "sending", "claim_token": token, "locked_until": current.Add(2 * time.Minute), "attempts": gorm.Expr("attempts + 1")})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}
			claimed = true
			return tx.First(&n, candidate.ID).Error
		})
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if !claimed {
			continue
		}
		// 整个渠道调用限制为 30 秒，领取有效期大于发送上限。
		sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = s.Senders[n.Channel].Send(sendCtx, notification.Message{ID: fmt.Sprint(n.ID), Recipient: n.Recipient, Subject: n.Subject, Content: n.Content})
		cancel()
		updates := map[string]any{"claim_token": "", "locked_until": nil, "last_error": ""}
		if err == nil {
			updates["status"] = "sent"
			updates["sent_at"] = time.Now().UTC()
		} else {
			updates["status"] = "failed"
			updates["last_error"] = "通知发送失败，请检查渠道配置和服务状态"
			var failure *notification.DeliveryError
			permanent := false
			if errors.As(err, &failure) {
				permanent = failure.Permanent
				updates["last_error"] = failure.Summary
			}

			if n.Attempts < 4 && !permanent {
				delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute}
				updates["status"] = "pending"
				updates["next_retry_at"] = time.Now().UTC().Add(delays[n.Attempts-1])
			}
		}
		// 退出取消后保留领取状态，由下次启动超时恢复，避免无界延长退出。
		if ctx.Err() != nil {
			return ctx.Err()
		}
		result := s.Orm.WithContext(ctx).Model(&models.SystemNotification{}).Where("id = ? AND status = ? AND claim_token = ?", n.ID, "sending", token).Updates(updates)
		if result.Error != nil {
			failures = append(failures, result.Error)
		}
	}
	return errors.Join(failures...)
}
