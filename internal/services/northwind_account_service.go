package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/array/banking-api/internal/integrations/northwind"
	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories"
	"github.com/google/uuid"
)

var (
	ErrExternalAccountValidationFailed = errors.New("external account validation failed")
	ErrExternalAccountAlreadyExists    = errors.New("external account already registered")
	ErrExternalAccountNotFound         = errors.New("external account not found")
)

// NorthwindAccountService handles external account registration and validation
type NorthwindAccountService struct {
	client *northwind.Client
	repo   repositories.NorthwindExternalAccountRepositoryInterface
	logger *slog.Logger
}

// NewNorthwindAccountService creates a new NorthWind account service
func NewNorthwindAccountService(
	client *northwind.Client,
	repo repositories.NorthwindExternalAccountRepositoryInterface,
	logger *slog.Logger,
) *NorthwindAccountService {
	return &NorthwindAccountService{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

// ValidateAndRegisterRequest represents a request to validate and register an external account
type ValidateAndRegisterRequest struct {
	AccountHolderName string `json:"account_holder_name" validate:"required"`
	AccountNumber     string `json:"account_number" validate:"required"`
	RoutingNumber     string `json:"routing_number" validate:"required"`
	InstitutionName   string `json:"institution_name,omitempty"`
}

// ValidateAndRegisterResponse represents the response from validation and registration
type ValidateAndRegisterResponse struct {
	Account    *models.NorthwindExternalAccount     `json:"account"`
	Validation *northwind.AccountValidationResponse `json:"validation"`
}

// ValidateAndRegister validates an external account with NorthWind and stores it locally
func (s *NorthwindAccountService) ValidateAndRegister(ctx context.Context, userID uuid.UUID, req ValidateAndRegisterRequest) (*ValidateAndRegisterResponse, error) {
	// Check if already registered
	existing, err := s.repo.FindByAccountAndRouting(userID, req.AccountNumber, req.RoutingNumber)
	if err == nil && existing != nil {
		if existing.Validated {
			return &ValidateAndRegisterResponse{
				Account: existing,
				Validation: &northwind.AccountValidationResponse{
					Valid:         true,
					AccountNumber: existing.AccountNumber,
					RoutingNumber: existing.RoutingNumber,
					Message:       "Account already registered and validated",
				},
			}, nil
		}
	}

	// Call NorthWind to validate
	validationResp, err := s.client.ValidateAccount(ctx, northwind.AccountValidationRequest{
		AccountNumber: req.AccountNumber,
		RoutingNumber: req.RoutingNumber,
	})
	if err != nil {
		s.logger.Error("NorthWind account validation failed", "error", err, "account_number", req.AccountNumber)
		return nil, fmt.Errorf("northwind validation error: %w", err)
	}

	if !validationResp.Valid {
		return &ValidateAndRegisterResponse{
			Validation: validationResp,
		}, ErrExternalAccountValidationFailed
	}

	// Upsert: if we found an existing unvalidated record, update it
	now := time.Now()
	if existing != nil {
		existing.AccountHolderName = req.AccountHolderName
		existing.Validated = true
		existing.ValidationTime = &now
		if req.InstitutionName != "" {
			existing.InstitutionName = &req.InstitutionName
		}
		if validationResp.InstitutionName != "" {
			existing.InstitutionName = &validationResp.InstitutionName
		}
		if err := s.repo.Update(existing); err != nil {
			return nil, fmt.Errorf("failed to update external account: %w", err)
		}
		return &ValidateAndRegisterResponse{
			Account:    existing,
			Validation: validationResp,
		}, nil
	}

	// Create new record
	institutionName := req.InstitutionName
	if validationResp.InstitutionName != "" {
		institutionName = validationResp.InstitutionName
	}
	var instPtr *string
	if institutionName != "" {
		instPtr = &institutionName
	}

	account := &models.NorthwindExternalAccount{
		UserID:            &userID,
		AccountHolderName: req.AccountHolderName,
		AccountNumber:     req.AccountNumber,
		RoutingNumber:     req.RoutingNumber,
		InstitutionName:   instPtr,
		Validated:         true,
		ValidationTime:    &now,
	}

	if err := s.repo.Create(account); err != nil {
		return nil, fmt.Errorf("failed to create external account: %w", err)
	}

	s.logger.Info("External account registered", "account_id", account.ID, "user_id", userID)

	return &ValidateAndRegisterResponse{
		Account:    account,
		Validation: validationResp,
	}, nil
}

// ListRegisteredAccounts returns the user's registered external accounts
func (s *NorthwindAccountService) ListRegisteredAccounts(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.NorthwindExternalAccount, int64, error) {
	return s.repo.GetByUserID(userID, offset, limit)
}

// ListAccessibleAccounts returns accessible accounts from NorthWind API (passthrough)
func (s *NorthwindAccountService) ListAccessibleAccounts(ctx context.Context) ([]northwind.ExternalAccount, error) {
	return s.client.ListAccounts(ctx, 100, 0, "", "")
}
