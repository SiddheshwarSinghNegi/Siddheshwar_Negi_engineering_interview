package services

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories/repository_mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestRegulatorService_CalculateBackoff(t *testing.T) {
	svc := &RegulatorService{
		retryInitialSeconds: 2,
		retryMaxSeconds:     60,
	}

	tests := []struct {
		attempt    int
		minSeconds float64
		maxSeconds float64
	}{
		{1, 1.0, 3.0},    // ~2s base
		{2, 2.0, 6.0},    // ~4s base
		{3, 5.0, 10.0},   // ~8s base
		{4, 10.0, 20.0},  // ~16s base
		{5, 20.0, 40.0},  // ~32s base
		{6, 40.0, 73.0},  // ~64s -> capped at 60
		{10, 40.0, 73.0}, // large attempt -> still capped
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			// Run multiple times due to jitter
			for i := 0; i < 20; i++ {
				backoff := svc.calculateBackoff(tt.attempt)
				seconds := backoff.Seconds()
				if seconds < tt.minSeconds {
					t.Errorf("attempt %d: backoff %v (%.2fs) below minimum %.2fs",
						tt.attempt, backoff, seconds, tt.minSeconds)
				}
				if seconds > tt.maxSeconds {
					t.Errorf("attempt %d: backoff %v (%.2fs) above maximum %.2fs",
						tt.attempt, backoff, seconds, tt.maxSeconds)
				}
			}
		})
	}
}

func TestRegulatorService_BackoffCap(t *testing.T) {
	svc := &RegulatorService{
		retryInitialSeconds: 2,
		retryMaxSeconds:     60,
	}

	// Even at very high attempt counts, should never exceed max + jitter
	for attempt := 1; attempt <= 20; attempt++ {
		backoff := svc.calculateBackoff(attempt)
		// Max is 60s + 20% jitter = 72s maximum theoretical
		if backoff > 73*time.Second {
			t.Errorf("attempt %d: backoff %v exceeds cap", attempt, backoff)
		}
		if backoff < 1*time.Second {
			t.Errorf("attempt %d: backoff %v below minimum 1s", attempt, backoff)
		}
	}
}

func TestRegulatorService_BackoffIsExponential(t *testing.T) {
	svc := &RegulatorService{
		retryInitialSeconds: 2,
		retryMaxSeconds:     120, // Higher cap to avoid capping during test
	}

	// Verify that backoff generally increases
	var prevMedian float64
	for attempt := 1; attempt <= 5; attempt++ {
		var total float64
		runs := 100
		for i := 0; i < runs; i++ {
			total += svc.calculateBackoff(attempt).Seconds()
		}
		median := total / float64(runs)

		if attempt > 1 && median <= prevMedian {
			t.Errorf("attempt %d: median backoff %.2fs should be greater than attempt %d: %.2fs",
				attempt, median, attempt-1, prevMedian)
		}
		prevMedian = median
	}
}

func makeTestNorthwindTransfer(t *testing.T) *models.NorthwindTransfer {
	t.Helper()
	return &models.NorthwindTransfer{
		ID:                  uuid.New(),
		NorthwindTransferID: uuid.New(),
		Amount:              decimal.NewFromFloat(100.50),
		Currency:            "USD",
		Direction:           "outbound",
		TransferType:        "ach",
		Status:              models.NWTransferStatusCompleted,
	}
}

func TestRegulatorService_CreateAndSendNotification_HTTP200_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifRepo := repository_mocks.NewMockRegulatorNotificationRepositoryInterface(ctrl)
	attemptRepo := repository_mocks.NewMockRegulatorNotificationAttemptRepositoryInterface(ctrl)
	transfer := makeTestNorthwindTransfer(t)

	notifRepo.EXPECT().ExistsForTransferAndStatus(transfer.ID, models.NWTransferStatusCompleted).Return(false, nil)
	notifRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(n *models.RegulatorNotification) error {
		n.ID = uuid.New()
		return nil
	}).Times(1)
	notifRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(n *models.RegulatorNotification) error {
		if !n.Delivered {
			t.Error("expected Delivered=true after 200")
		}
		return nil
	}).Times(1)
	attemptRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)

	svc := NewRegulatorService(
		server.URL,
		2, 60,
		notifRepo, attemptRepo,
		slog.Default(),
		server.Client(),
	)
	ctx := context.Background()
	err := svc.CreateAndSendNotification(ctx, transfer, models.NWTransferStatusCompleted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegulatorService_CreateAndSendNotification_HTTP500_SchedulesRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifRepo := repository_mocks.NewMockRegulatorNotificationRepositoryInterface(ctrl)
	attemptRepo := repository_mocks.NewMockRegulatorNotificationAttemptRepositoryInterface(ctrl)
	transfer := makeTestNorthwindTransfer(t)

	notifRepo.EXPECT().ExistsForTransferAndStatus(transfer.ID, models.NWTransferStatusFailed).Return(false, nil)
	notifRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(n *models.RegulatorNotification) error {
		n.ID = uuid.New()
		return nil
	}).Times(1)
	notifRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(n *models.RegulatorNotification) error {
		if n.Delivered {
			t.Error("expected Delivered=false after 500")
		}
		if n.NextAttemptAt == nil {
			t.Error("expected NextAttemptAt set for retry")
		}
		return nil
	}).Times(1)
	attemptRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)

	svc := NewRegulatorService(
		server.URL,
		2, 60,
		notifRepo, attemptRepo,
		slog.Default(),
		server.Client(),
	)
	ctx := context.Background()
	err := svc.CreateAndSendNotification(ctx, transfer, models.NWTransferStatusFailed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegulatorService_CreateAndSendNotification_Idempotency_SkipsIfExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifRepo := repository_mocks.NewMockRegulatorNotificationRepositoryInterface(ctrl)
	attemptRepo := repository_mocks.NewMockRegulatorNotificationAttemptRepositoryInterface(ctrl)
	transfer := makeTestNorthwindTransfer(t)

	notifRepo.EXPECT().ExistsForTransferAndStatus(transfer.ID, models.NWTransferStatusCompleted).Return(true, nil)
	notifRepo.EXPECT().Create(gomock.Any()).Times(0)
	attemptRepo.EXPECT().Create(gomock.Any()).Times(0)

	svc := NewRegulatorService(
		"http://localhost:9999/webhook",
		2, 60,
		notifRepo, attemptRepo,
		slog.Default(),
		nil,
	)
	ctx := context.Background()
	err := svc.CreateAndSendNotification(ctx, transfer, models.NWTransferStatusCompleted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegulatorService_RetryOnce_DeliversPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifRepo := repository_mocks.NewMockRegulatorNotificationRepositoryInterface(ctrl)
	attemptRepo := repository_mocks.NewMockRegulatorNotificationAttemptRepositoryInterface(ctrl)

	payload := []byte(`{"event_id":"e1","transfer_id":"t1","status":"COMPLETED"}`)
	notif := models.RegulatorNotification{
		ID:             uuid.New(),
		TransferID:     uuid.New(),
		TerminalStatus: models.NWTransferStatusCompleted,
		Delivered:      false,
		AttemptCount:   0,
		Payload:        payload,
	}
	now := time.Now()
	notif.NextAttemptAt = &now

	notifRepo.EXPECT().GetPendingNotifications(20).Return([]models.RegulatorNotification{notif}, nil)
	notifRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(n *models.RegulatorNotification) error {
		if !n.Delivered {
			t.Error("expected Delivered=true after 200")
		}
		return nil
	}).Times(1)
	attemptRepo.EXPECT().Create(gomock.Any()).Return(nil).Times(1)

	svc := NewRegulatorService(
		server.URL,
		2, 60,
		notifRepo, attemptRepo,
		slog.Default(),
		server.Client(),
	)
	ctx := context.Background()
	svc.RetryOnce(ctx)
}
