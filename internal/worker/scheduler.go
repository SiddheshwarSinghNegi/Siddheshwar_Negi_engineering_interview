package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/array/banking-api/internal/services"
)

// Scheduler runs NorthWind transfer polling and regulator notification retries in a single loop.
// One ticker drives both job types to avoid multiple timer goroutines.
type Scheduler struct {
	polling   *services.NorthwindPollingService
	regulator *services.RegulatorService
	interval  time.Duration
	logger    *slog.Logger
}

// NewScheduler creates a unified scheduler for NorthWind polling and regulator retries
func NewScheduler(
	polling *services.NorthwindPollingService,
	regulator *services.RegulatorService,
	interval time.Duration,
	logger *slog.Logger,
) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{
		polling:   polling,
		regulator: regulator,
		interval:  interval,
		logger:    logger,
	}
}

// Start runs the scheduler loop until ctx is cancelled.
// Each tick: (1) poll NorthWind for transfer status updates, (2) retry pending regulator notifications.
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("Unified worker scheduler started", "interval", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Unified worker scheduler stopping")
			return
		case <-ticker.C:
			s.polling.PollOnce(ctx)
			s.regulator.RetryOnce(ctx)
		}
	}
}
