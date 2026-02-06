package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories"
	"github.com/google/uuid"
)

// RegulatorService handles webhook notifications to the regulator
type RegulatorService struct {
	webhookURL          string
	retryInitialSeconds int
	retryMaxSeconds     int
	notifRepo           repositories.RegulatorNotificationRepositoryInterface
	attemptRepo         repositories.RegulatorNotificationAttemptRepositoryInterface
	httpClient          *http.Client
	logger              *slog.Logger
}

// NewRegulatorService creates a new regulator service
func NewRegulatorService(
	webhookURL string,
	retryInitialSeconds int,
	retryMaxSeconds int,
	notifRepo repositories.RegulatorNotificationRepositoryInterface,
	attemptRepo repositories.RegulatorNotificationAttemptRepositoryInterface,
	logger *slog.Logger,
) *RegulatorService {
	return &RegulatorService{
		webhookURL:          webhookURL,
		retryInitialSeconds: retryInitialSeconds,
		retryMaxSeconds:     retryMaxSeconds,
		notifRepo:           notifRepo,
		attemptRepo:         attemptRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// CreateAndSendNotification creates a notification record and immediately attempts delivery
func (s *RegulatorService) CreateAndSendNotification(ctx context.Context, transfer *models.NorthwindTransfer, terminalStatus string) error {
	// Idempotency guard: check if notification already exists for this transfer+status
	exists, err := s.notifRepo.ExistsForTransferAndStatus(transfer.ID, terminalStatus)
	if err != nil {
		return fmt.Errorf("failed to check notification existence: %w", err)
	}
	if exists {
		s.logger.Info("Notification already exists for transfer, skipping",
			"transfer_id", transfer.ID,
			"status", terminalStatus,
		)
		return nil
	}

	// Build webhook payload
	amount, _ := transfer.Amount.Float64()
	payload := models.RegulatorWebhookPayload{
		EventID:             uuid.New().String(),
		TransferID:          transfer.ID.String(),
		NorthwindTransferID: transfer.NorthwindTransferID.String(),
		Status:              terminalStatus,
		Amount:              amount,
		Currency:            transfer.Currency,
		Direction:           transfer.Direction,
		TransferType:        transfer.TransferType,
		Timestamp:           time.Now().UTC().Format(time.RFC3339),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	now := time.Now()
	notification := &models.RegulatorNotification{
		TransferID:     transfer.ID,
		TerminalStatus: terminalStatus,
		Delivered:      false,
		AttemptCount:   0,
		NextAttemptAt:  &now, // Immediate first attempt
		Payload:        payloadBytes,
	}

	if err := s.notifRepo.Create(notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	s.logger.Info("Regulator notification created, attempting immediate delivery",
		"notification_id", notification.ID,
		"transfer_id", transfer.ID,
	)

	// Immediately attempt first delivery (meeting 60-second requirement)
	s.attemptDelivery(ctx, notification)

	return nil
}

// StartRetryLoop runs the background retry loop for undelivered notifications
func (s *RegulatorService) StartRetryLoop(ctx context.Context) {
	s.logger.Info("Regulator retry service started")
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Regulator retry service stopping")
			return
		case <-ticker.C:
			s.retryPendingNotifications(ctx)
		}
	}
}

func (s *RegulatorService) retryPendingNotifications(ctx context.Context) {
	notifications, err := s.notifRepo.GetPendingNotifications(20)
	if err != nil {
		s.logger.Error("Failed to fetch pending regulator notifications", "error", err)
		return
	}

	for i := range notifications {
		select {
		case <-ctx.Done():
			return
		default:
			s.attemptDelivery(ctx, &notifications[i])
		}
	}
}

func (s *RegulatorService) attemptDelivery(ctx context.Context, notification *models.RegulatorNotification) {
	now := time.Now()

	// Prepare HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(notification.Payload))
	if err != nil {
		s.recordAttempt(notification, nil, fmt.Sprintf("failed to create request: %v", err), "")
		s.scheduleRetry(notification)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event-ID", notification.ID.String())

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Warn("Regulator webhook delivery failed",
			"notification_id", notification.ID,
			"attempt", notification.AttemptCount+1,
			"error", err,
		)
		s.recordAttempt(notification, nil, err.Error(), "")
		s.scheduleRetry(notification)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	respBody := string(bodyBytes)
	// Truncate response body for storage
	if len(respBody) > 1000 {
		respBody = respBody[:1000]
	}

	httpStatus := resp.StatusCode

	if httpStatus >= 200 && httpStatus < 300 {
		// Success
		notification.Delivered = true
		notification.AttemptCount++
		notification.LastAttemptAt = &now
		notification.LastHTTPStatus = &httpStatus
		if notification.FirstAttemptAt == nil {
			notification.FirstAttemptAt = &now
		}
		notification.NextAttemptAt = nil
		notification.LastError = nil

		if err := s.notifRepo.Update(notification); err != nil {
			s.logger.Error("Failed to update notification after successful delivery", "error", err)
		}

		s.recordAttempt(notification, &httpStatus, "", respBody)

		s.logger.Info("Regulator notification delivered successfully",
			"notification_id", notification.ID,
			"transfer_id", notification.TransferID,
			"attempts", notification.AttemptCount,
		)
		return
	}

	// Non-success HTTP status
	errMsg := fmt.Sprintf("webhook returned HTTP %d", httpStatus)
	s.logger.Warn("Regulator webhook returned non-success status",
		"notification_id", notification.ID,
		"http_status", httpStatus,
		"attempt", notification.AttemptCount+1,
	)

	s.recordAttempt(notification, &httpStatus, errMsg, respBody)
	s.scheduleRetry(notification)
}

func (s *RegulatorService) recordAttempt(notification *models.RegulatorNotification, httpStatus *int, errMsg, respBody string) {
	attempt := &models.RegulatorNotificationAttempt{
		NotificationID: notification.ID,
		HTTPStatus:     httpStatus,
	}
	if errMsg != "" {
		attempt.Error = &errMsg
	}
	if respBody != "" {
		attempt.ResponseBody = &respBody
	}

	if err := s.attemptRepo.Create(attempt); err != nil {
		s.logger.Error("Failed to record notification attempt", "error", err)
	}
}

func (s *RegulatorService) scheduleRetry(notification *models.RegulatorNotification) {
	now := time.Now()
	notification.AttemptCount++
	notification.LastAttemptAt = &now
	if notification.FirstAttemptAt == nil {
		notification.FirstAttemptAt = &now
	}

	// Exponential backoff with jitter
	backoff := s.calculateBackoff(notification.AttemptCount)
	nextAttempt := now.Add(backoff)
	notification.NextAttemptAt = &nextAttempt

	if err := s.notifRepo.Update(notification); err != nil {
		s.logger.Error("Failed to schedule retry", "error", err)
	}

	s.logger.Info("Regulator notification retry scheduled",
		"notification_id", notification.ID,
		"attempt", notification.AttemptCount,
		"next_attempt_at", nextAttempt,
		"backoff", backoff,
	)
}

// calculateBackoff returns the backoff duration using exponential backoff with jitter
func (s *RegulatorService) calculateBackoff(attemptCount int) time.Duration {
	base := float64(s.retryInitialSeconds)
	max := float64(s.retryMaxSeconds)

	// Exponential: base * 2^(attempt-1)
	backoffSeconds := base * math.Pow(2, float64(attemptCount-1))

	// Cap at max
	if backoffSeconds > max {
		backoffSeconds = max
	}

	// Add jitter: +/- 20%
	jitter := backoffSeconds * 0.2 * (rand.Float64()*2 - 1) //nolint:gosec
	backoffSeconds += jitter

	if backoffSeconds < 1 {
		backoffSeconds = 1
	}

	return time.Duration(backoffSeconds * float64(time.Second))
}
