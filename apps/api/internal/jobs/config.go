package jobs

import (
	"math"
	"math/rand"
	"time"
)

// Config holds background job runtime settings.
type Config struct {
	MaxAttempts      int
	BaseBackoff      time.Duration
	MaxBackoff       time.Duration
	WorkerPoll       time.Duration
	SchedulerTick    time.Duration
	ReminderWindows  []int
	DefaultIngestMin int
	ProcessingLease  time.Duration
}

// DefaultConfig returns production-safe defaults.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:      5,
		BaseBackoff:      30 * time.Second,
		MaxBackoff:       30 * time.Minute,
		WorkerPoll:       2 * time.Second,
		SchedulerTick:    60 * time.Second,
		ReminderWindows:  []int{7, 3, 1},
		DefaultIngestMin: 360,
		ProcessingLease:  15 * time.Minute,
	}
}

// NextBackoff computes exponential backoff with jitter.
func (c Config) NextBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	exp := float64(c.BaseBackoff) * math.Pow(2, float64(attempt-1))
	if exp > float64(c.MaxBackoff) {
		exp = float64(c.MaxBackoff)
	}
	jitter := rand.Float64() * 0.25 * exp
	return time.Duration(exp + jitter)
}
