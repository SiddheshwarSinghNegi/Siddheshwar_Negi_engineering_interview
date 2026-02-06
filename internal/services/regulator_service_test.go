package services

import (
	"testing"
	"time"
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
