package repositories

import (
	"errors"
	"fmt"

	"github.com/array/banking-api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNorthwindTransferNotFound = errors.New("northwind transfer not found")
)

type northwindTransferRepository struct {
	db *gorm.DB
}

// NewNorthwindTransferRepository creates a new NorthWind transfer repository
func NewNorthwindTransferRepository(db *gorm.DB) NorthwindTransferRepositoryInterface {
	return &northwindTransferRepository{db: db}
}

func (r *northwindTransferRepository) Create(transfer *models.NorthwindTransfer) error {
	if transfer == nil {
		return errors.New("transfer cannot be nil")
	}
	if err := r.db.Create(transfer).Error; err != nil {
		return fmt.Errorf("failed to create northwind transfer: %w", err)
	}
	return nil
}

func (r *northwindTransferRepository) Update(transfer *models.NorthwindTransfer) error {
	if transfer == nil {
		return errors.New("transfer cannot be nil")
	}
	if err := r.db.Save(transfer).Error; err != nil {
		return fmt.Errorf("failed to update northwind transfer: %w", err)
	}
	return nil
}

func (r *northwindTransferRepository) GetByID(id uuid.UUID) (*models.NorthwindTransfer, error) {
	var transfer models.NorthwindTransfer
	if err := r.db.Where("id = ?", id).First(&transfer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNorthwindTransferNotFound
		}
		return nil, fmt.Errorf("failed to get northwind transfer: %w", err)
	}
	return &transfer, nil
}

func (r *northwindTransferRepository) GetByNorthwindTransferID(nwID uuid.UUID) (*models.NorthwindTransfer, error) {
	var transfer models.NorthwindTransfer
	if err := r.db.Where("northwind_transfer_id = ?", nwID).First(&transfer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNorthwindTransferNotFound
		}
		return nil, fmt.Errorf("failed to get northwind transfer by nw id: %w", err)
	}
	return &transfer, nil
}

func (r *northwindTransferRepository) GetByUserID(userID uuid.UUID, offset, limit int) ([]models.NorthwindTransfer, int64, error) {
	return r.GetByUserIDWithFilters(userID, "", "", "", offset, limit)
}

func (r *northwindTransferRepository) GetByUserIDWithFilters(userID uuid.UUID, status, direction, transferType string, offset, limit int) ([]models.NorthwindTransfer, int64, error) {
	var transfers []models.NorthwindTransfer
	var total int64

	query := r.db.Model(&models.NorthwindTransfer{}).Where("user_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if direction != "" {
		query = query.Where("direction = ?", direction)
	}
	if transferType != "" {
		query = query.Where("transfer_type = ?", transferType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count northwind transfers: %w", err)
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&transfers).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list northwind transfers: %w", err)
	}

	return transfers, total, nil
}

func (r *northwindTransferRepository) GetPendingTransfers(limit int) ([]models.NorthwindTransfer, error) {
	var transfers []models.NorthwindTransfer
	if err := r.db.Where("status IN ?", []string{models.NWTransferStatusPending, models.NWTransferStatusProcessing}).
		Order("created_at ASC").
		Limit(limit).
		Find(&transfers).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending northwind transfers: %w", err)
	}
	return transfers, nil
}
