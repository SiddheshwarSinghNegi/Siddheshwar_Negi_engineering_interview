package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/array/banking-api/internal/integrations/northwind"
	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrNWTransferValidationFailed = errors.New("transfer validation failed")
	ErrNWTransferInsufficientBal  = errors.New("insufficient balance in source account")
	ErrNWTransferInitiateFailed   = errors.New("failed to initiate transfer with northwind")
	ErrNWTransferNotFound         = errors.New("northwind transfer not found")
)

// NorthwindTransferService handles external transfer operations
type NorthwindTransferService struct {
	client       *northwind.Client
	transferRepo repositories.NorthwindTransferRepositoryInterface
	logger       *slog.Logger
}

// NewNorthwindTransferService creates a new NorthWind transfer service
func NewNorthwindTransferService(
	client *northwind.Client,
	transferRepo repositories.NorthwindTransferRepositoryInterface,
	logger *slog.Logger,
) *NorthwindTransferService {
	return &NorthwindTransferService{
		client:       client,
		transferRepo: transferRepo,
		logger:       logger,
	}
}

// CreateTransferRequest represents a request to create an external transfer
type CreateTransferRequest struct {
	Amount             float64                      `json:"amount" validate:"required,gt=0"`
	Currency           string                       `json:"currency" validate:"required"`
	Description        string                       `json:"description,omitempty"`
	Direction          string                       `json:"direction" validate:"required,oneof=INBOUND OUTBOUND"`
	TransferType       string                       `json:"transfer_type" validate:"required"`
	ReferenceNumber    string                       `json:"reference_number" validate:"required"`
	ScheduledDate      string                       `json:"scheduled_date,omitempty"`
	SourceAccount      CreateTransferAccountDetails `json:"source_account" validate:"required"`
	DestinationAccount CreateTransferAccountDetails `json:"destination_account" validate:"required"`
}

// CreateTransferAccountDetails represents account details in a transfer request
type CreateTransferAccountDetails struct {
	AccountHolderName string `json:"account_holder_name" validate:"required"`
	AccountNumber     string `json:"account_number" validate:"required"`
	RoutingNumber     string `json:"routing_number,omitempty"`
	InstitutionName   string `json:"institution_name,omitempty"`
}

// CreateTransferResponse represents the response from creating a transfer
type CreateTransferResponse struct {
	Transfer          *models.NorthwindTransfer   `json:"transfer"`
	NorthwindResponse *northwind.TransferResponse `json:"northwind_response,omitempty"`
}

// CreateTransfer validates, checks balance, initiates a transfer via NorthWind, and stores it locally
func (s *NorthwindTransferService) CreateTransfer(ctx context.Context, userID uuid.UUID, req CreateTransferRequest) (*CreateTransferResponse, error) {
	// Build NorthWind transfer request
	nwReq := northwind.TransferRequest{
		Amount:             req.Amount,
		Currency:           req.Currency,
		Description:        req.Description,
		Direction:          req.Direction,
		TransferType:       req.TransferType,
		ReferenceNumber:    req.ReferenceNumber,
		ScheduledDate:      req.ScheduledDate,
		SourceAccount:      toNWAccountDetails(req.SourceAccount),
		DestinationAccount: toNWAccountDetails(req.DestinationAccount),
	}

	// Step 1: Validate transfer with NorthWind
	validationResp, err := s.client.ValidateTransfer(ctx, nwReq)
	if err != nil {
		s.logger.Warn("NorthWind transfer validation call failed", "error", err)
		// Non-blocking: if validation endpoint fails, proceed to initiate
	} else if validationResp != nil && !validationResp.Valid {
		// Check for severity=error issues
		for _, issue := range validationResp.Issues {
			if issue.Severity == "error" {
				return nil, fmt.Errorf("%w: %s", ErrNWTransferValidationFailed, issue.Message)
			}
		}
	}

	// Step 2: Check balance for source account (best effort)
	balance, err := s.client.GetAccountBalance(ctx, req.SourceAccount.AccountNumber)
	if err != nil {
		s.logger.Warn("Balance check failed, proceeding with initiation", "error", err)
	} else if balance != nil && balance.AvailableBalance < req.Amount {
		return nil, fmt.Errorf("%w: available=%.2f, requested=%.2f",
			ErrNWTransferInsufficientBal, balance.AvailableBalance, req.Amount)
	}

	// Step 3: Initiate transfer with NorthWind
	nwResp, err := s.client.InitiateTransfer(ctx, nwReq)
	if err != nil {
		s.logger.Error("NorthWind transfer initiation failed", "error", err)
		return nil, fmt.Errorf("%w: %v", ErrNWTransferInitiateFailed, err)
	}

	// Step 4: Store locally
	nwTransferID, err := uuid.Parse(nwResp.TransferID)
	if err != nil {
		s.logger.Error("Failed to parse northwind transfer ID", "transfer_id", nwResp.TransferID, "error", err)
		nwTransferID = uuid.New() // fallback
	}

	transfer := &models.NorthwindTransfer{
		UserID:                   &userID,
		NorthwindTransferID:      nwTransferID,
		Direction:                req.Direction,
		TransferType:             req.TransferType,
		Amount:                   decimal.NewFromFloat(req.Amount),
		Currency:                 req.Currency,
		ReferenceNumber:          req.ReferenceNumber,
		SourceAccountNumber:      req.SourceAccount.AccountNumber,
		DestinationAccountNumber: req.DestinationAccount.AccountNumber,
		Status:                   northwind.MapStatus(nwResp.Status),
	}

	if req.Description != "" {
		transfer.Description = &req.Description
	}
	if req.SourceAccount.RoutingNumber != "" {
		transfer.SourceRoutingNumber = &req.SourceAccount.RoutingNumber
	}
	if req.SourceAccount.AccountHolderName != "" {
		transfer.SourceAccountHolderName = &req.SourceAccount.AccountHolderName
	}
	if req.DestinationAccount.RoutingNumber != "" {
		transfer.DestinationRoutingNumber = &req.DestinationAccount.RoutingNumber
	}
	if req.DestinationAccount.AccountHolderName != "" {
		transfer.DestinationAccountHolderName = &req.DestinationAccount.AccountHolderName
	}

	transfer.InitiatedDate = northwind.ParseRFC3339Optional(nwResp.InitiatedDate)
	transfer.ProcessingDate = northwind.ParseRFC3339Optional(nwResp.ProcessingDate)
	transfer.ExpectedCompletionDate = northwind.ParseRFC3339Optional(nwResp.ExpectedCompletionDate)
	transfer.CompletedDate = northwind.ParseRFC3339Optional(nwResp.CompletedDate)

	if nwResp.ScheduledDate != "" {
		transfer.ScheduledDate = northwind.ParseRFC3339Optional(nwResp.ScheduledDate)
	} else if req.ScheduledDate != "" {
		transfer.ScheduledDate = northwind.ParseRFC3339Optional(req.ScheduledDate)
	}

	if nwResp.Fee != nil {
		fee := decimal.NewFromFloat(*nwResp.Fee)
		transfer.Fee = &fee
	}
	if nwResp.ExchangeRate != nil {
		rate := decimal.NewFromFloat(*nwResp.ExchangeRate)
		transfer.ExchangeRate = &rate
	}
	if nwResp.ErrorCode != "" {
		transfer.ErrorCode = &nwResp.ErrorCode
	}
	if nwResp.ErrorMessage != "" {
		transfer.ErrorMessage = &nwResp.ErrorMessage
	}

	if err := s.transferRepo.Create(transfer); err != nil {
		s.logger.Error("Failed to store transfer locally", "error", err)
		return nil, fmt.Errorf("failed to store transfer: %w", err)
	}

	s.logger.Info("Transfer initiated and stored",
		"local_id", transfer.ID,
		"northwind_id", nwTransferID,
		"status", transfer.Status,
	)

	return &CreateTransferResponse{
		Transfer:          transfer,
		NorthwindResponse: nwResp,
	}, nil
}

// GetTransfer retrieves a local NorthWind transfer by ID
func (s *NorthwindTransferService) GetTransfer(ctx context.Context, userID uuid.UUID, transferID uuid.UUID) (*models.NorthwindTransfer, error) {
	transfer, err := s.transferRepo.GetByID(transferID)
	if err != nil {
		return nil, err
	}
	// Verify ownership
	if transfer.UserID != nil && *transfer.UserID != userID {
		return nil, ErrNWTransferNotFound
	}
	return transfer, nil
}

// ListTransfers lists the user's NorthWind transfers with optional filters
func (s *NorthwindTransferService) ListTransfers(ctx context.Context, userID uuid.UUID, status, direction, transferType string, offset, limit int) ([]models.NorthwindTransfer, int64, error) {
	return s.transferRepo.GetByUserIDWithFilters(userID, status, direction, transferType, offset, limit)
}

// CancelTransfer cancels a transfer via NorthWind
func (s *NorthwindTransferService) CancelTransfer(ctx context.Context, userID uuid.UUID, transferID uuid.UUID, reason string) (*models.NorthwindTransfer, error) {
	transfer, err := s.GetTransfer(ctx, userID, transferID)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.CancelTransfer(ctx, transfer.NorthwindTransferID.String(), reason)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel transfer: %w", err)
	}

	transfer.Status = northwind.MapStatus(resp.Status)
	if resp.ErrorCode != "" {
		transfer.ErrorCode = &resp.ErrorCode
	}
	if resp.ErrorMessage != "" {
		transfer.ErrorMessage = &resp.ErrorMessage
	}

	if err := s.transferRepo.Update(transfer); err != nil {
		return nil, fmt.Errorf("failed to update transfer after cancel: %w", err)
	}

	return transfer, nil
}

// ReverseTransfer reverses a transfer via NorthWind
func (s *NorthwindTransferService) ReverseTransfer(ctx context.Context, userID uuid.UUID, transferID uuid.UUID, reason, description string) (*models.NorthwindTransfer, error) {
	transfer, err := s.GetTransfer(ctx, userID, transferID)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.ReverseTransfer(ctx, transfer.NorthwindTransferID.String(), reason, description)
	if err != nil {
		return nil, fmt.Errorf("failed to reverse transfer: %w", err)
	}

	transfer.Status = northwind.MapStatus(resp.Status)
	if resp.ErrorCode != "" {
		transfer.ErrorCode = &resp.ErrorCode
	}
	if resp.ErrorMessage != "" {
		transfer.ErrorMessage = &resp.ErrorMessage
	}

	if err := s.transferRepo.Update(transfer); err != nil {
		return nil, fmt.Errorf("failed to update transfer after reverse: %w", err)
	}

	return transfer, nil
}

func toNWAccountDetails(d CreateTransferAccountDetails) northwind.AccountDetails {
	return northwind.AccountDetails{
		AccountHolderName: d.AccountHolderName,
		AccountNumber:     d.AccountNumber,
		RoutingNumber:     d.RoutingNumber,
		InstitutionName:   d.InstitutionName,
	}
}

