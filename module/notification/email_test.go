package notification

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io"
	"mime"
	"net"
	"net/http/httptest"
	"net/mail"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestEmailSendTLS 验证真实 TLS 握手及中文邮件编码，不发送外部邮件。
func TestEmailSendTLS(t *testing.T) {
	certificateServer := httptest.NewTLSServer(nil)
	certificate := certificateServer.TLS.Certificates[0]
	certificateServer.Close()
	parsed, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(parsed)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	result := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			result <- ""
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		c := textproto.NewConn(conn)
		_ = c.PrintfLine("220 localhost SMTP")
		for {
			line, err := c.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"):
				_ = c.PrintfLine("250 localhost")
			case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
				_ = c.PrintfLine("250 OK")
			case line == "DATA":
				_ = c.PrintfLine("354 send data")
				body, err := io.ReadAll(c.DotReader())
				if err != nil {
					result <- ""
					return
				}
				result <- string(body)
				_ = c.PrintfLine("250 accepted")
			case line == "QUIT":
				_ = c.PrintfLine("221 bye")
				return
			default:
				_ = c.PrintfLine("500 unsupported")
			}
		}
	}()
	_, portText, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portText)
	sender, err := NewEmailSender(EmailConfig{Host: "127.0.0.1", Port: port, From: "sender@example.com", TLSMode: "tls", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	sender.tlsConfig.RootCAs = roots
	content := strings.Repeat("请执行计划。", 30) + "\n下一行"
	if err := sender.Send(context.Background(), Message{ID: "1", Recipient: "user@example.com", Subject: "计划到期提醒", Content: content}); err != nil {
		t.Fatal(err)
	}
	select {
	case raw := <-result:
		m, err := mail.ReadMessage(strings.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		subject, err := new(mime.WordDecoder).DecodeHeader(m.Header.Get("Subject"))
		if err != nil || subject != "计划到期提醒" {
			t.Fatalf("标题编码错误: %s %v", subject, err)
		}
		decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, m.Body))
		if err != nil {
			t.Fatal(err)
		}
		if strings.ReplaceAll(string(decoded), "\r\n", "\n") != content {
			t.Fatal("中文正文不一致")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("未收到测试邮件")
	}
}

// TestEmailRejectsUnsafeHeaders 验证邮件头注入和不安全配置会被拒绝。
func TestEmailRejectsUnsafeHeaders(t *testing.T) {
	c := EmailConfig{Host: "localhost", Port: 465, From: "sender@example.com", TLSMode: "tls"}
	sender, err := NewEmailSender(c)
	if err != nil {
		t.Fatal(err)
	}
	if err = sender.Send(context.Background(), Message{Recipient: "user@example.com", Subject: "标题\r\nBcc: other@example.com"}); err == nil {
		t.Fatal("应拒绝邮件头注入")
	}
	c.TLSMode = "none"
	if _, err = NewEmailSender(c); err == nil {
		t.Fatal("应拒绝明文 SMTP")
	}
}

// TestEmailTimeout 验证服务器不响应握手时按上下文超时退出。
func TestEmailTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			_, _ = io.Copy(io.Discard, conn)
		}
		close(done)
	}()
	_, text, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(text)
	sender, err := NewEmailSender(EmailConfig{Host: "127.0.0.1", Port: port, From: "sender@example.com", TLSMode: "tls", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err = sender.Send(ctx, Message{Recipient: "user@example.com"}); err == nil {
		t.Fatal("握手应超时")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("超时后连接未关闭")
	}
}
