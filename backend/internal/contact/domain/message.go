package domain

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

const (
	MaxNameLength    = 100
	MaxEmailLength   = 160
	MaxPhoneLength   = 30
	MaxMessageLength = 2000
	MinMessageLength = 20
)

var phonePattern = regexp.MustCompile(`^[+0-9()-][+0-9() -]{5,28}[+0-9()-]$`)

type Subject string

const (
	SubjectProduct      Subject = "Informasi produk"
	SubjectDistribution Subject = "Kerja sama dan distribusi"
	SubjectGeneral      Subject = "Pertanyaan umum"
)

func (subject Subject) Valid() bool {
	switch subject {
	case SubjectProduct, SubjectDistribution, SubjectGeneral:
		return true
	default:
		return false
	}
}

// Message is the canonical contact payload accepted by the application.
type Message struct {
	Name           string
	Email          string
	Phone          string
	Subject        Subject
	Body           string
	PrivacyConsent bool
	BotField       string
}

func (message Message) Validate() error {
	if message.BotField != "" {
		return errors.New("honeypot must be empty")
	}
	if message.Name != strings.TrimSpace(message.Name) ||
		len([]rune(message.Name)) < 2 ||
		len([]rune(message.Name)) > MaxNameLength {
		return errors.New("invalid name")
	}
	if message.Email != strings.TrimSpace(message.Email) ||
		len(message.Email) > MaxEmailLength {
		return errors.New("invalid email")
	}
	address, err := mail.ParseAddress(message.Email)
	if err != nil || address.Address != message.Email || !strings.Contains(message.Email, ".") {
		return errors.New("invalid email")
	}
	if message.Phone != "" &&
		(message.Phone != strings.TrimSpace(message.Phone) ||
			len([]rune(message.Phone)) < 7 ||
			len([]rune(message.Phone)) > MaxPhoneLength ||
			!phonePattern.MatchString(message.Phone)) {
		return errors.New("invalid phone")
	}
	if !message.Subject.Valid() {
		return errors.New("invalid subject")
	}
	if message.Body != strings.TrimSpace(message.Body) ||
		len([]rune(message.Body)) < MinMessageLength ||
		len([]rune(message.Body)) > MaxMessageLength {
		return errors.New("invalid message")
	}
	if !message.PrivacyConsent {
		return errors.New("privacy consent is required")
	}
	return nil
}

// RetentionDeleteAt applies the approved maximum 12-month retention window.
func RetentionDeleteAt(consentAt time.Time) time.Time {
	return consentAt.UTC().AddDate(0, 12, 0)
}
