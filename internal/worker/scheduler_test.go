package worker

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/array/banking-api/internal/models"
	"github.com/array/banking-api/internal/repositories/repository_mocks"
	"github.com/array/banking-api/internal/services"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler_NilLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifRepo := repository_mocks.NewMockRegulatorNotificationRepositoryInterface(ctrl)
	notifRepo.EXPECT().GetPendingNotifications(20).Return([]models.RegulatorNotification{}, nil).AnyTimes()
	attemptRepo := repository_mocks.NewMockRegulatorNotificationAttemptRepositoryInterface(ctrl)
	regulator := services.NewRegulatorService("http://localhost", 2, 60, notifRepo, attemptRepo, nil, nil)

	transferRepo := repository_mocks.NewMockNorthwindTransferRepositoryInterface(ctrl)
	transferRepo.EXPECT().GetPendingTransfers(50).Return([]models.NorthwindTransfer{}, nil).AnyTimes()
	polling := services.NewNorthwindPollingService(nil, transferRepo, regulator, time.Hour, nil)

	sched := NewScheduler(polling, regulator, time.Second, nil)
	require.NotNil(t, sched)
	assert.NotNil(t, sched.logger)
}

func TestScheduler_Start_StopsOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifRepo := repository_mocks.NewMockRegulatorNotificationRepositoryInterface(ctrl)
	notifRepo.EXPECT().GetPendingNotifications(20).Return([]models.RegulatorNotification{}, nil).AnyTimes()
	attemptRepo := repository_mocks.NewMockRegulatorNotificationAttemptRepositoryInterface(ctrl)
	regulator := services.NewRegulatorService("http://localhost", 2, 60, notifRepo, attemptRepo, slog.Default(), nil)

	transferRepo := repository_mocks.NewMockNorthwindTransferRepositoryInterface(ctrl)
	transferRepo.EXPECT().GetPendingTransfers(50).Return([]models.NorthwindTransfer{}, nil).AnyTimes()
	polling := services.NewNorthwindPollingService(nil, transferRepo, regulator, time.Hour, slog.Default())

	sched := NewScheduler(polling, regulator, 10*time.Second, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		sched.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Start returned as expected
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancel")
	}
}

func TestScheduler_Start_RunsOneTickThenStops(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifRepo := repository_mocks.NewMockRegulatorNotificationRepositoryInterface(ctrl)
	notifRepo.EXPECT().GetPendingNotifications(20).Return([]models.RegulatorNotification{}, nil).AnyTimes()
	attemptRepo := repository_mocks.NewMockRegulatorNotificationAttemptRepositoryInterface(ctrl)
	regulator := services.NewRegulatorService("http://localhost", 2, 60, notifRepo, attemptRepo, slog.Default(), nil)

	transferRepo := repository_mocks.NewMockNorthwindTransferRepositoryInterface(ctrl)
	transferRepo.EXPECT().GetPendingTransfers(50).Return([]models.NorthwindTransfer{}, nil).AnyTimes()
	polling := services.NewNorthwindPollingService(nil, transferRepo, regulator, time.Hour, slog.Default())

	sched := NewScheduler(polling, regulator, 5*time.Millisecond, slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		sched.Start(ctx)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Start did not return after cancel")
	}
}
