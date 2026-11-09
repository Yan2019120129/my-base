package notification

import "context"

// Message 保存通知渠道需要的消息快照。
type Message struct {
	ID        string
	Recipient string
	Subject   string
	Content   string
}

// Sender 定义通知发送能力，成功只表示渠道已接受消息。
type Sender interface {
	Send(context.Context, Message) error
}

// DeliveryError 提供渠道无关的失败摘要与重试决策，避免业务层依赖协议错误。
type DeliveryError struct {
	Summary   string
	Permanent bool
}

// Error 返回不含凭据及消息正文的失败摘要。
func (e *DeliveryError) Error() string { return e.Summary }
