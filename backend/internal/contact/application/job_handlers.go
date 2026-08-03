package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type Notification struct {
	MessageID string
	Name      string
	Email     string
	Subject   string
	Message   string
}

type SideEffectStore interface {
	LoadNotification(context.Context, string) (Notification, error)
	RecordReceivedActivity(context.Context, string) error
}

type EmailSender interface {
	SendContactNotification(context.Context, Notification, string) error
}

type JobHandlers struct {
	store SideEffectStore
	email EmailSender
}

func NewJobHandlers(store SideEffectStore, email EmailSender) (*JobHandlers, error) {
	if store == nil || email == nil {
		return nil, errors.New("contact job handlers require store and email sender")
	}
	return &JobHandlers{store: store, email: email}, nil
}

func (handlers *JobHandlers) Email(ctx context.Context, payload json.RawMessage, reference string) error {
	messageID, err := contactMessageID(payload)
	if err != nil {
		return err
	}
	notification, err := handlers.store.LoadNotification(ctx, messageID)
	if err != nil {
		return fmt.Errorf("load contact notification: %w", err)
	}
	if err := handlers.email.SendContactNotification(ctx, notification, reference); err != nil {
		return fmt.Errorf("send contact notification: %w", err)
	}
	return nil
}

func (handlers *JobHandlers) Activity(ctx context.Context, payload json.RawMessage) error {
	messageID, err := contactMessageID(payload)
	if err != nil {
		return err
	}
	if err := handlers.store.RecordReceivedActivity(ctx, messageID); err != nil {
		return fmt.Errorf("record contact activity: %w", err)
	}
	return nil
}

func contactMessageID(payload json.RawMessage) (string, error) {
	var value struct {
		ContactMessageID string `json:"contactMessageId"`
	}
	if err := json.Unmarshal(payload, &value); err != nil || value.ContactMessageID == "" {
		return "", errors.New("invalid contact job payload")
	}
	return value.ContactMessageID, nil
}
