package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RegulatorNotification represents a webhook notification to the regulator for a terminal transfer
type RegulatorNotification struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	TransferID     uuid.UUID       `gorm:"type:uuid;not null" json:"transfer_id"`
	TerminalStatus string          `gorm:"type:text;not null" json:"terminal_status"`
	Delivered      bool            `gorm:"not null;default:false" json:"delivered"`
	AttemptCount   int             `gorm:"not null;default:0" json:"attempt_count"`
	FirstAttemptAt *time.Time      `json:"first_attempt_at,omitempty"`
	LastAttemptAt  *time.Time      `json:"last_attempt_at,omitempty"`
	NextAttemptAt  *time.Time      `json:"next_attempt_at,omitempty"`
	LastHTTPStatus *int            `json:"last_http_status,omitempty"`
	LastError      *string         `json:"last_error,omitempty"`
	Payload        json.RawMessage `gorm:"type:jsonb;not null" json:"payload"`
	CreatedAt      time.Time       `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"not null" json:"updated_at"`
}

// TableName returns the table name for RegulatorNotification
func (r *RegulatorNotification) TableName() string {
	return "regulator_notifications"
}

// BeforeCreate hook for RegulatorNotification
func (r *RegulatorNotification) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	now := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	if r.UpdatedAt.IsZero() {
		r.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook for RegulatorNotification
func (r *RegulatorNotification) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = time.Now()
	return nil
}

// RegulatorNotificationAttempt records a single delivery attempt for audit proof
type RegulatorNotificationAttempt struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	NotificationID uuid.UUID  `gorm:"type:uuid;not null" json:"notification_id"`
	AttemptedAt    time.Time  `gorm:"not null" json:"attempted_at"`
	HTTPStatus     *int       `json:"http_status,omitempty"`
	Error          *string    `json:"error,omitempty"`
	ResponseBody   *string    `gorm:"type:text" json:"response_body,omitempty"`
}

// TableName returns the table name for RegulatorNotificationAttempt
func (r *RegulatorNotificationAttempt) TableName() string {
	return "regulator_notification_attempts"
}

// BeforeCreate hook for RegulatorNotificationAttempt
func (r *RegulatorNotificationAttempt) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	if r.AttemptedAt.IsZero() {
		r.AttemptedAt = time.Now()
	}
	return nil
}

// RegulatorWebhookPayload is the payload sent to the regulator webhook
type RegulatorWebhookPayload struct {
	EventID            string  `json:"event_id"`
	TransferID         string  `json:"transfer_id"`
	NorthwindTransferID string `json:"northwind_transfer_id"`
	Status             string  `json:"status"`
	Amount             float64 `json:"amount"`
	Currency           string  `json:"currency"`
	Direction          string  `json:"direction"`
	TransferType       string  `json:"transfer_type"`
	Timestamp          string  `json:"timestamp"`
}
