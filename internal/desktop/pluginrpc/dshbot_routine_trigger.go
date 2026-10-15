package pluginrpc

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// This is a get-or-create mutation, not an ordinary describe/read endpoint.
// The original Host shares one persistent token across its routine catalog.
// Call only after a separate explicit user confirmation. Never retry here.
func (c *Client) DshbotObtainRoutineTriggerToken(ctx context.Context) ([]byte, error) {
	var result struct {
		Token string `json:"token"`
	}
	if err := c.Call(ctx, "dshbot", "bot/trigger-token", map[string]any{}, &result); err != nil {
		// Do not expose response/error bodies that might contain a credential.
		return nil, fmt.Errorf("could not obtain the trigger credential; it may already exist; no retry was sent")
	}
	if result.Token == "" || len(result.Token) > 4096 || strings.ContainsAny(result.Token, "\r\n\x00") {
		return nil, fmt.Errorf("Host returned an unusable trigger credential; no retry was sent")
	}
	return []byte(result.Token), nil
}

var routineTriggerIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

func DshbotRoutineTriggerURL(origin, routineID string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || !routineTriggerIDPattern.MatchString(routineID) || routineID == "." || routineID == ".." {
		return "", fmt.Errorf("cannot represent this Host/routine as a safe trigger URL")
	}
	u.Path = "/dshbot-hook/routine/" + routineID
	return u.String(), nil
}
