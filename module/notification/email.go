package notification

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

// EmailConfig 定义 SMTP 配置，不依赖应用配置包。
type EmailConfig struct {
	Host                              string
	Port                              int
	Username, Password, From, TLSMode string
	Timeout                           time.Duration
}

// EmailSender 通过经过证书验证的 TLS 连接发送邮件。
type EmailSender struct {
	config    EmailConfig
	tlsConfig *tls.Config
}

// NewEmailSender 校验配置并创建邮件渠道。
func NewEmailSender(c EmailConfig) (*EmailSender, error) {
	if c.Host == "" || c.Port < 1 || c.Port > 65535 {
		return nil, errors.New("SMTP 地址或端口无效")
	}
	if c.TLSMode != "starttls" && c.TLSMode != "tls" {
		return nil, errors.New("SMTP TLS 模式必须为 starttls 或 tls")
	}
	if strings.ContainsAny(c.From, "\r\n") {
		return nil, errors.New("发件地址无效")
	}
	if _, err := mail.ParseAddress(c.From); err != nil {
		return nil, err
	}
	if c.Timeout <= 0 {
		c.Timeout = 15 * time.Second
	}
	return &EmailSender{config: c, tlsConfig: &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}}, nil
}

// Send 发送 UTF-8 邮件，并用连接截止时间限制 SMTP 操作。
func (s *EmailSender) Send(ctx context.Context, m Message) (result error) {
	defer func() {
		if result == nil {
			return
		}
		var existing *DeliveryError
		if errors.As(result, &existing) {
			return
		}
		failure := &DeliveryError{Summary: "邮件发送失败，请检查 SMTP 配置和服务状态"}
		var smtpErr *textproto.Error
		var networkErr net.Error
		if errors.As(result, &smtpErr) {
			failure.Permanent = smtpErr.Code >= 500 && smtpErr.Code < 600
			failure.Summary = fmt.Sprintf("SMTP 返回状态码 %d，请检查接收地址和渠道配置", smtpErr.Code)
		} else if errors.Is(result, context.DeadlineExceeded) || (errors.As(result, &networkErr) && networkErr.Timeout()) {
			failure.Summary = "邮件发送超时"
		}
		result = failure
	}()
	if strings.ContainsAny(m.Recipient+m.Subject+m.ID, "\r\n") {
		return &DeliveryError{Summary: "邮件头包含非法换行", Permanent: true}
	}
	recipient, err := mail.ParseAddress(m.Recipient)
	if err != nil {
		return &DeliveryError{Summary: "接收邮箱无效", Permanent: true}
	}
	from, _ := mail.ParseAddress(s.config.From)
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	address := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline, _ := ctx.Deadline()
	if err = conn.SetDeadline(deadline); err != nil {
		return err
	}
	rawConn := conn
	stop := context.AfterFunc(ctx, func() { _ = rawConn.Close() })
	defer stop()
	tlsConfig := s.tlsConfig.Clone()
	if s.config.TLSMode == "tls" {
		secure := tls.Client(conn, tlsConfig)
		if err = secure.HandshakeContext(ctx); err != nil {
			return err
		}
		conn = secure
	}
	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if s.config.TLSMode == "starttls" {
		if err = client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if s.config.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)); err != nil {
			return err
		}
	}
	if err = client.Mail(from.Address); err != nil {
		return err
	}
	if err = client.Rcpt(recipient.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	body := strings.ReplaceAll(strings.ReplaceAll(m.Content, "\r\n", "\n"), "\n", "\r\n")
	_, err = fmt.Fprintf(writer, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: base64\r\n\r\n%s\r\n", from.String(), recipient.String(), mime.QEncoding.Encode("UTF-8", m.Subject), encodeBody(body))
	if err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	// DATA 已被接受后不因 QUIT 失败触发重复发送。
	_ = client.Quit()
	return nil
}

// encodeBody 按 MIME 行长度限制编码正文。
func encodeBody(body string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(body))
	var out strings.Builder
	for len(encoded) > 76 {
		out.WriteString(encoded[:76])
		out.WriteString("\r\n")
		encoded = encoded[76:]
	}
	out.WriteString(encoded)
	return out.String()
}
