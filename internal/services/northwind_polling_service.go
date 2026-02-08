package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/array/banking-api/internal/integrations/northwind"
	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories"
)

// NorthwindPollingService periodically polls NorthWind for transfer status updates
type NorthwindPollingService struct {
	client       *northwind.Client
	transferRepo repositories.NorthwindTransferRepositoryInterface
	regulatorSvc *RegulatorService
	pollInterval time.Duration
	logger       *slog.Logger
}

// NewNorthwindPollingService creates a new polling service
func NewNorthwindPollingService(
	client *northwind.Client,
	transferRepo repositories.NorthwindTransferRepositoryInterface,
	regulatorSvc *RegulatorService,
	pollInterval time.Duration,
	logger *slog.Logger,
) *NorthwindPollingService {
	return &NorthwindPollingService{
		client:       client,
		transferRepo: transferRepo,
		regulatorSvc: regulatorSvc,
		pollInterval: pollInterval,
		logger:       logger,
	}
}

// Start begins the polling loop. Blocks until ctx is cancelled.
func (s *NorthwindPollingService) Start(ctx context.Context) {
	s.logger.Info("NorthWind polling service started", "interval", s.pollInterval)
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("NorthWind polling service stopping")
			return
		case <-ticker.C:
			s.PollOnce(ctx)
		}
	}
}

// PollOnce runs one transfer status poll cycle. Used by the unified worker scheduler.
func (s *NorthwindPollingService) PollOnce(ctx context.Context) {
	s.pollPendingTransfers(ctx)
}

func (s *NorthwindPollingService) pollPendingTransfers(ctx context.Context) {
	transfers, err := s.transferRepo.GetPendingTransfers(50)
	if err != nil {
		s.logger.Error("Failed to fetch pending NorthWind transfers", "error", err)
		return
	}

	if len(transfers) == 0 {
		return
	}

	s.logger.Info("Polling NorthWind for transfer status updates", "count", len(transfers))

	for i := range transfers {
		select {
		case <-ctx.Done():
			return
		default:
			s.checkTransferStatus(ctx, &transfers[i])
		}
	}
}

func (s *NorthwindPollingService) checkTransferStatus(ctx context.Context, transfer *models.NorthwindTransfer) {
	resp, err := s.client.GetTransferStatus(ctx, transfer.NorthwindTransferID.String())
	if err != nil {
		s.logger.Warn("Failed to get transfer status from NorthWind",
			"northwind_id", transfer.NorthwindTransferID,
			"error", err,
		)
		return
	}

	newStatus := northwind.MapStatus(resp.Status)
	if newStatus == transfer.Status {
		return // No change
	}

	oldStatus := transfer.Status
	transfer.Status = newStatus

	// Update optional fields from response
	transfer.ProcessingDate = northwind.ParseRFC3339Optional(resp.ProcessingDate)
	transfer.CompletedDate = northwind.ParseRFC3339Optional(resp.CompletedDate)
	transfer.ExpectedCompletionDate = northwind.ParseRFC3339Optional(resp.ExpectedCompletionDate)

	if resp.ErrorCode != "" {
		transfer.ErrorCode = &resp.ErrorCode
	}
	if resp.ErrorMessage != "" {
		transfer.ErrorMessage = &resp.ErrorMessage
	}

	if err := s.transferRepo.Update(transfer); err != nil {
		s.logger.Error("Failed to update transfer status",
			"transfer_id", transfer.ID,
			"error", err,
		)
		return
	}

	s.logger.Info("Transfer status updated",
		"transfer_id", transfer.ID,
		"northwind_id", transfer.NorthwindTransferID,
		"old_status", oldStatus,
		"new_status", newStatus,
	)

	// If terminal state, trigger regulator notification
	if newStatus == models.NWTransferStatusCompleted || newStatus == models.NWTransferStatusFailed {
		s.logger.Info("Transfer reached terminal state, creating regulator notification",
			"transfer_id", transfer.ID,
			"status", newStatus,
		)
		if err := s.regulatorSvc.CreateAndSendNotification(ctx, transfer, newStatus); err != nil {
			s.logger.Error("Failed to create regulator notification",
				"transfer_id", transfer.ID,
				"error", err,
			)
		}
	}
}
