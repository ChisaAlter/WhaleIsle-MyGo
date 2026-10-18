package pluginrpc

import (
	"context"
	"fmt"
	"strings"
)

// DshbotReadMark uses the original monotonic mark-read protocol, not catalog CAS.
type DshbotReadMark struct {
	ID         string `json:"id"`
	ThroughSeq int    `json:"throughSeq"`
}

func (c *Client) DshbotMarkRead(ctx context.Context, marks []DshbotReadMark) (DshbotSettingsView, error) {
	if len(marks) == 0 || len(marks) > 200 {
		return DshbotSettingsView{}, fmt.Errorf("invalid read markers")
	}
	for _, m := range marks {
		if strings.TrimSpace(m.ID) == "" || m.ThroughSeq < 0 {
			return DshbotSettingsView{}, fmt.Errorf("invalid read marker")
		}
	}
	var result struct {
		View DshbotSettingsView `json:"view"`
	}
	if err := c.Call(ctx, "dshbot", "bot/mark-read", map[string]any{"marks": marks}, &result); err != nil {
		return DshbotSettingsView{}, err
	}
	for _, mark := range marks {
		found := false
		for _, item := range result.View.Value.Items {
			if item.ID == mark.ID {
				found = item.ReadInitialized && item.LastSeenSeq >= mark.ThroughSeq
				break
			}
		}
		if !found {
			return result.View, fmt.Errorf("Host did not confirm read marker for %s", mark.ID)
		}
	}
	return result.View, nil
}
