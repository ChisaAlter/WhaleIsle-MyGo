package config

import (
	"fmt"
	"strconv"
	"strings"
)

// RestartPreferences is the committed, bounded ordinary crash-retry policy.
// It does not control manual restart or the separate plugin recovery attempt.
type RestartPreferences struct {
	AutoRestart bool
	MaxAttempts int
	BaseDelayMs int
}

func RestartPreferencesFrom(c Config) RestartPreferences {
	return RestartPreferences{c.HarnessAutoRestart, c.HarnessRestartMaxAttempts, c.HarnessRestartBaseDelayMs}
}

// ParseRestartPreferences rejects an invalid draft before any config mutation.
// Unlike load-time normalization, a user save must never silently accept a
// different retry count or delay. The bounds match normalize in config.go.
func ParseRestartPreferences(enabled bool, attempts, delay string) (RestartPreferences, error) {
	max, err := strconv.Atoi(strings.TrimSpace(attempts))
	if err != nil || max < 1 || max > 10 {
		return RestartPreferences{}, fmt.Errorf("invalid Harness restart attempts (1-10)")
	}
	base, err := strconv.Atoi(strings.TrimSpace(delay))
	if err != nil || base < 500 || base > 30000 {
		return RestartPreferences{}, fmt.Errorf("invalid Harness restart base delay (500-30000 ms)")
	}
	return RestartPreferences{enabled, max, base}, nil
}

// Apply participates in the caller's existing atomic Store.Update transaction.
// It changes no other config fields and does not interact with a process.
func (p RestartPreferences) Apply(c *Config) {
	c.HarnessAutoRestart = p.AutoRestart
	c.HarnessRestartMaxAttempts = p.MaxAttempts
	c.HarnessRestartBaseDelayMs = p.BaseDelayMs
}
