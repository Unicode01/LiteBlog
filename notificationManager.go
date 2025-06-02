package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

type NotifyManager struct {
	notifyType                   NotifyType // available types: email
	minDelayBetweenNotifications int        // seconds
	lastNotificationTime         int64      // Unix timestamp
	notifyTimeout                int        // seconds
}

func NewNotifyManager(notifyType NotifyType) *NotifyManager {
	nm := &NotifyManager{
		notifyType:                   notifyType,
		minDelayBetweenNotifications: 60,
		lastNotificationTime:         0,
		notifyTimeout:                60, // 60s
	}
	return nm
}

func (nm *NotifyManager) Notify(title, content string) error {
	now := time.Now().Unix()
	if now-nm.lastNotificationTime < int64(nm.minDelayBetweenNotifications) {
		return fmt.Errorf("notification skipped: too soon after last notification")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(nm.notifyTimeout)*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		nm.lastNotificationTime = time.Now().Unix()
		err := nm.notifyType.SendNotification(title, content)
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("notification timed out: %w", ctx.Err())
	}
}

type NotifyType interface {
	SendNotification(title, content string) error
}

type NotifyTypeSMTP struct {
	SmtpServer string // e.g. smtp.gmail.com:587
	SmtpUser   string
	SmtpPass   string
	FromEmail  string
	ToEmail    []string
}

func (n *NotifyTypeSMTP) SendNotification(title, content string) error {
	// send email using smtp
	docHTML := fmt.Sprintf("New notification: \n%s\n\n%s", title, content)
	return n.SendEmail(title, docHTML, false)
}

func (n *NotifyTypeSMTP) SendEmail(subject, body string, isHTML bool) error {
	// 创建认证信息
	auth := smtp.PlainAuth("", n.SmtpUser, n.SmtpPass, strings.Split(n.SmtpServer, ":")[0])

	// 构建邮件内容
	var contentType string
	if isHTML {
		contentType = "text/html; charset=UTF-8"
	} else {
		contentType = "text/plain; charset=UTF-8"
	}

	// 编码主题（支持中文）
	encodedSubject := "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="

	// 构建邮件头部
	headers := make(map[string]string)
	headers["From"] = n.FromEmail
	headers["To"] = strings.Join(n.ToEmail, ", ")
	headers["Subject"] = encodedSubject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = contentType

	// 拼接完整邮件内容
	message := new(bytes.Buffer)
	for k, v := range headers {
		fmt.Fprintf(message, "%s: %s\r\n", k, v)
	}
	message.WriteString("\r\n" + body)

	// 发送邮件
	return smtp.SendMail(
		n.SmtpServer,
		auth,
		n.SmtpUser,
		n.ToEmail,
		message.Bytes(),
	)
}

type NotifyTypeTelegramBot struct {
	BotToken string
	ChatID   string
}

func (n *NotifyTypeTelegramBot) SendNotification(title, content string) error {
	// send telegram message using bot API
	return n.sendMessage(fmt.Sprintf("New notification: \n%s\n\n%s", title, content))
}

func (n *NotifyTypeTelegramBot) sendMessage(text string) error {
	// send message to telegram chat
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s", n.BotToken, n.ChatID, url.QueryEscape(text))
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// fmt.Printf("Telegram response: %s\n,body: %s\n", resp.Status, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message: %s", resp.Status)
	}
	return nil
}
