package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// RecoveryConfig is the Host client-connection plugin's bootstrap policy.
// Values are milliseconds, matching its schema rather than Go nanoseconds.
type RecoveryConfig struct {
	BackoffBaseMS            int64
	BackoffFactor            float64
	BackoffMaxMS             int64
	GenerationReadyWarnMS    int64
	GenerationReadyTimeoutMS int64
}

func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{500, 2, 10000, 3000, 15000}
}

func (c RecoveryConfig) validate() error {
	for name, value := range map[string]int64{"backoffBaseMs": c.BackoffBaseMS, "backoffMaxMs": c.BackoffMaxMS, "generationReadyWarnMs": c.GenerationReadyWarnMS, "generationReadyTimeoutMs": c.GenerationReadyTimeoutMS} {
		if value < 1 || value > math.MaxInt32 {
			return fmt.Errorf("invalid Host connection recovery %s", name)
		}
	}
	if math.IsNaN(c.BackoffFactor) || math.IsInf(c.BackoffFactor, 0) || c.BackoffFactor < 1 {
		return errors.New("invalid Host connection recovery backoffFactor")
	}
	return nil
}

func (c RecoveryConfig) backoff() Backoff {
	return Backoff{Initial: time.Duration(min(c.BackoffBaseMS, c.BackoffMaxMS)) * time.Millisecond, Max: time.Duration(c.BackoffMaxMS) * time.Millisecond, Factor: c.BackoffFactor}
}

// ReadConnectionRecovery reads only the existing JSON bootstrap global. No
// plugin configuration file, credentials value or script execution is involved.
func (c *Client) ReadConnectionRecovery(ctx context.Context) (RecoveryConfig, error) {
	body, err := c.readBootDocument(ctx)
	if err != nil {
		return RecoveryConfig{}, err
	}
	return parseConnectionRecovery(string(body))
}

func parseConnectionRecovery(document string) (RecoveryConfig, error) {
	config := DefaultRecoveryConfig()
	tokenizer := html.NewTokenizer(strings.NewReader(document))
	script := false
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if tokenizer.Err() != io.EOF {
				return RecoveryConfig{}, errors.New("invalid Host boot document")
			}
			return config, nil
		case html.StartTagToken:
			token := tokenizer.Token()
			script = token.Data == "script"
			for _, attr := range token.Attr {
				if attr.Key == "src" {
					script = false
				}
			}
		case html.EndTagToken:
			script = false
		case html.TextToken:
			if !script {
				continue
			}
			text := strings.TrimSpace(string(tokenizer.Text()))
			const prefix = `globalThis["__DSH_CONNECTION_RECOVERY__"]`
			if !strings.HasPrefix(text, prefix) {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(text, prefix))
			if !strings.HasPrefix(value, "=") {
				return RecoveryConfig{}, errors.New("invalid Host connection recovery")
			}
			value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value[1:]), ";"))
			var values map[string]json.RawMessage
			if json.Unmarshal([]byte(value), &values) != nil || values == nil {
				return RecoveryConfig{}, errors.New("invalid Host connection recovery")
			}
			config = DefaultRecoveryConfig()
			for name, raw := range values {
				var number float64
				switch name {
				case "backoffBaseMs", "backoffFactor", "backoffMaxMs", "generationReadyWarnMs", "generationReadyTimeoutMs":
					if string(raw) == "null" || json.Unmarshal(raw, &number) != nil || math.IsNaN(number) || math.IsInf(number, 0) {
						return RecoveryConfig{}, fmt.Errorf("invalid Host connection recovery %s", name)
					}
				default:
					continue
				}
				if name == "backoffFactor" {
					config.BackoffFactor = number
					continue
				}
				if number < 1 || number > math.MaxInt32 || math.Trunc(number) != number {
					return RecoveryConfig{}, fmt.Errorf("invalid Host connection recovery %s", name)
				}
				switch name {
				case "backoffBaseMs":
					config.BackoffBaseMS = int64(number)
				case "backoffMaxMs":
					config.BackoffMaxMS = int64(number)
				case "generationReadyWarnMs":
					config.GenerationReadyWarnMS = int64(number)
				case "generationReadyTimeoutMs":
					config.GenerationReadyTimeoutMS = int64(number)
				}
			}
			if err := config.validate(); err != nil {
				return RecoveryConfig{}, err
			}
		}
	}
}
