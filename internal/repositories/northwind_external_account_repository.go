package repositories

import (
	"errors"
	"fmt"

	"github.com/array/banking-api/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNorthwindExternalAccountNotFound = errors.New("northwind external account not found")
)

type northwindExternalAccountRepository struct {
	db *gorm.DB
}

// NewNorthwindExternalAccountRepository creates a new NorthWind external account repository
func NewNorthwindExternalAccountRepository(db *gorm.DB) NorthwindExternalAccountRepositoryInterface {
	return &northwindExternalAccountRepository{db: db}
}

func (r *northwindExternalAccountRepository) Create(account *models.NorthwindExternalAccount) error {
	if account == nil {
		return errors.New("account cannot be nil")
	}
	if err := r.db.Create(account).Error; err != nil {
		if isDuplicateKeyError(err) {
			return fmt.Errorf("external account already registered: %w", err)
		}
		return fmt.Errorf("failed to create northwind external account: %w", err)
	}
	return nil
}

func (r *northwindExternalAccountRepository) GetByID(id uuid.UUID) (*models.NorthwindExternalAccount, error) {
	var account models.NorthwindExternalAccount
	if err := r.db.Where("id = ?", id).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNorthwindExternalAccountNotFound
		}
		return nil, fmt.Errorf("failed to get northwind external account: %w", err)
	}
	return &account, nil
}

func (r *northwindExternalAccountRepository) GetByUserID(userID uuid.UUID, offset, limit int) ([]models.NorthwindExternalAccount, int64, error) {
	var accounts []models.NorthwindExternalAccount
	var total int64

	query := r.db.Model(&models.NorthwindExternalAccount{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count northwind external accounts: %w", err)
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&accounts).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list northwind external accounts: %w", err)
	}

	return accounts, total, nil
}

func (r *northwindExternalAccountRepository) FindByAccountAndRouting(userID uuid.UUID, accountNumber, routingNumber string) (*models.NorthwindExternalAccount, error) {
	var account models.NorthwindExternalAccount
	if err := r.db.Where("user_id = ? AND account_number = ? AND routing_number = ?", userID, accountNumber, routingNumber).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNorthwindExternalAccountNotFound
		}
		return nil, fmt.Errorf("failed to find northwind external account: %w", err)
	}
	return &account, nil
}

func (r *northwindExternalAccountRepository) Update(account *models.NorthwindExternalAccount) error {
	if account == nil {
		return errors.New("account cannot be nil")
	}
	if err := r.db.Save(account).Error; err != nil {
		return fmt.Errorf("failed to update northwind external account: %w", err)
	}
	return nil
}
