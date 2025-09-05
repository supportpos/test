package model

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

type Notification struct {
	ID        int64     `json:"id"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationInput struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Message   string `json:"message"`
}

func (n NotificationInput) Validate() error {
	if _, err := mail.ParseAddress(n.Sender); err != nil {
		return errors.New("invalid sender email")
	}
	if _, err := mail.ParseAddress(n.Recipient); err != nil {
		return errors.New("invalid recipient email")
	}
	if strings.TrimSpace(n.Message) == "" {
		return errors.New("message required")
	}
	return nil
}

func MaskEmail(e string) string {
	parts := strings.Split(e, "@")
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	if len(local) > 2 {
		local = local[:1] + "***" + local[len(local)-1:]
	} else {
		local = "***"
	}
	return local + "@" + parts[1]
}
