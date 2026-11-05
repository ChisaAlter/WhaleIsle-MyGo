package imonboard

import "strings"

// Host errors contain the platform's public code/reason. Retain that useful
// feedback while excluding the precise credential values submitted by this UI.
func publicOperationError(err error, payload map[string]any) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	for _, key := range []string{"appSecret", "clientSecret", "secret", "token", "botToken", "appToken", "verifyCode"} {
		if value, ok := payload[key].(string); ok && value != "" {
			message = strings.ReplaceAll(message, value, "[redacted]")
		}
	}
	if len(message) > 4096 {
		message = message[:4096] + "…"
	}
	return message
}
