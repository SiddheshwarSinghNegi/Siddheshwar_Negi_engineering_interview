package repositories

import (
	"errors"
	"fmt"
	"time"

	"github.com/array/banking-api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrRegulatorNotificationNotFound = errors.New("regulator notification not found")
)

type regulatorNotificationRepository struct {
	db *gorm.DB
}

// NewRegulatorNotificationRepository creates a new regulator notification repository
func NewRegulatorNotificationRepository(db *gorm.DB) RegulatorNotificationRepositoryInterface {
	return &regulatorNotificationRepository{db: db}
}

func (r *regulatorNotificationRepository) Create(notification *models.RegulatorNotification) error {
	if notification == nil {
		return errors.New("notification cannot be nil")
	}
	if err := r.db.Create(notification).Error; err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("notification already exists for this transfer and status: %w", err)
		}
		return fmt.Errorf("failed to create regulator notification: %w", err)
	}
	return nil
}

func (r *regulatorNotificationRepository) Update(notification *models.RegulatorNotification) error {
	if notification == nil {
		return errors.New("notification cannot be nil")
	}
	if err := r.db.Save(notification).Error; err != nil {
		return fmt.Errorf("failed to update regulator notification: %w", err)
	}
	return nil
}

func (r *regulatorNotificationRepository) GetByID(id uuid.UUID) (*models.RegulatorNotification, error) {
	var notification models.RegulatorNotification
	if err := r.db.Where("id = ?", id).First(&notification).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRegulatorNotificationNotFound
		}
		return nil, fmt.Errorf("failed to get regulator notification: %w", err)
	}
	return &notification, nil
}

func (r *regulatorNotificationRepository) GetPendingNotifications(limit int) ([]models.RegulatorNotification, error) {
	var notifications []models.RegulatorNotification
	now := time.Now()
	if err := r.db.Where("delivered = ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?)", false, now).
		Order("created_at ASC").
		Limit(limit).
		Find(&notifications).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending regulator notifications: %w", err)
	}
	return notifications, nil
}

func (r *regulatorNotificationRepository) ExistsForTransferAndStatus(transferID uuid.UUID, terminalStatus string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.RegulatorNotification{}).
		Where("transfer_id = ? AND terminal_status = ?", transferID, terminalStatus).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check regulator notification existence: %w", err)
	}
	return count > 0, nil
}

// --- Notification Attempt Repository ---

type regulatorNotificationAttemptRepository struct {
	db *gorm.DB
}

// NewRegulatorNotificationAttemptRepository creates a new regulator notification attempt repository
func NewRegulatorNotificationAttemptRepository(db *gorm.DB) RegulatorNotificationAttemptRepositoryInterface {
	return &regulatorNotificationAttemptRepository{db: db}
}

func (r *regulatorNotificationAttemptRepository) Create(attempt *models.RegulatorNotificationAttempt) error {
	if attempt == nil {
		return errors.New("attempt cannot be nil")
	}
	if err := r.db.Create(attempt).Error; err != nil {
		return fmt.Errorf("failed to create notification attempt: %w", err)
	}
	return nil
}

func (r *regulatorNotificationAttemptRepository) GetByNotificationID(notificationID uuid.UUID) ([]models.RegulatorNotificationAttempt, error) {
	var attempts []models.RegulatorNotificationAttempt
	if err := r.db.Where("notification_id = ?", notificationID).
		Order("attempted_at ASC").
		Find(&attempts).Error; err != nil {
		return nil, fmt.Errorf("failed to get notification attempts: %w", err)
	}
	return attempts, nil
}
