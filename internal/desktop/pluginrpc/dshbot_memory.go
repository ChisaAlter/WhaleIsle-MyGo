package pluginrpc

import (
	"context"
	"fmt"
	"strings"
)

type DshbotMemoryTrack struct {
	Entries  []string `json:"entries"`
	Text     string   `json:"text"`
	Revision string   `json:"revision"`
}
type DshbotMemoryLimits struct {
	Bot  int `json:"bot"`
	User int `json:"user"`
}
type DshbotMemoryResult struct {
	BotID  string             `json:"botId"`
	Bot    DshbotMemoryTrack  `json:"bot"`
	User   DshbotMemoryTrack  `json:"user"`
	Limits DshbotMemoryLimits `json:"limits"`
}

func (v DshbotMemoryResult) Track(track string) DshbotMemoryTrack {
	if track == "user" {
		return v.User
	}
	return v.Bot
}
func (v DshbotMemoryResult) Limit(track string) int {
	n := v.Limits.Bot
	if track == "user" {
		n = v.Limits.User
	}
	return min(n, 64000)
}
func CloneDshbotMemory(v DshbotMemoryResult) DshbotMemoryResult {
	v.Bot.Entries = append([]string(nil), v.Bot.Entries...)
	v.User.Entries = append([]string(nil), v.User.Entries...)
	return v
}
func ValidateDshbotMemory(botID string, v DshbotMemoryResult) error {
	if botID == "" || v.BotID != botID || v.Bot.Revision == "" || v.User.Revision == "" || v.Limits.Bot <= 0 || v.Limits.User <= 0 {
		return fmt.Errorf("Host memory identity, revision or limits unavailable")
	}
	return nil
}
func (c *Client) DshbotMemory(ctx context.Context, botID string) (DshbotMemoryResult, error) {
	var out DshbotMemoryResult
	if strings.TrimSpace(botID) == "" {
		return out, fmt.Errorf("select a Bot first")
	}
	if err := c.Call(ctx, "dshbot", "memory/get", map[string]any{"id": botID}, &out); err != nil {
		return out, fmt.Errorf("could not read Bot memory; refresh before editing")
	}
	return out, ValidateDshbotMemory(botID, out)
}

// The original UI trims each entry and joins '- ' bullets without a final LF.
// ECMAScript trim includes FEFF but not NEL; preserve internal newlines verbatim.
func trimBotMemory(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return r >= 0x2000 && r <= 0x200a || strings.ContainsRune("\t\n\v\f\r \u00a0\u1680\u2028\u2029\u202f\u205f\u3000\ufeff", r)
	})
}
func DshbotMemoryText(entries []string) string {
	rows := make([]string, 0, len(entries))
	for _, e := range entries {
		if e = trimBotMemory(e); e != "" {
			rows = append(rows, "- "+e)
		}
	}
	return strings.Join(rows, "\n")
}

type DshbotMemoryCommand struct {
	BotID, SessionID, Track, MemoryRevision, Text string
	Limit                                         int
}
type DshbotMemoryReceipt struct {
	BotID          string `json:"botId"`
	Track          string `json:"track"`
	MemoryRevision string `json:"memoryRevision"`
	DshbotMemoryTrack
	View DshbotSettingsView `json:"view"`
}

func ValidateDshbotMemoryCommand(cmd DshbotMemoryCommand) error {
	if cmd.BotID == "" || cmd.SessionID == "" || (cmd.Track != "bot" && cmd.Track != "user") || cmd.MemoryRevision == "" || cmd.Limit <= 0 {
		return fmt.Errorf("memory target, track, revision or limit unavailable")
	}
	if WhaleTextLength(cmd.Text) > min(cmd.Limit, 64000) {
		return fmt.Errorf("memory exceeds this track's Host limit (UTF-16)")
	}
	return nil
}
func ValidateDshbotMemoryReceipt(cmd DshbotMemoryCommand, out DshbotMemoryReceipt) error {
	if out.BotID != cmd.BotID || out.Track != cmd.Track || out.Text != cmd.Text || out.MemoryRevision == "" || out.MemoryRevision != out.Revision || out.View.NS != "dsh-bot" {
		return fmt.Errorf("memory result is not the confirmed replacement; refresh before retrying")
	}
	matches := 0
	for _, item := range out.View.Value.Items {
		if item.ID == cmd.BotID && item.Kind != "room" && item.SessionID == cmd.SessionID {
			matches++
		}
	}
	if matches != 1 {
		return fmt.Errorf("Bot session identity changed; refresh before retrying")
	}
	return nil
}
func (c *Client) DshbotReplaceMemory(ctx context.Context, cmd DshbotMemoryCommand) (DshbotMemoryReceipt, error) {
	var out DshbotMemoryReceipt
	if err := ValidateDshbotMemoryCommand(cmd); err != nil {
		return out, err
	}
	// CAS belongs to this exact memory track, not the unrelated catalog revision.
	if err := c.Call(ctx, "dshbot", "memory/replace", map[string]any{"id": cmd.BotID, "track": cmd.Track, "memoryRevision": cmd.MemoryRevision, "text": cmd.Text}, &out); err != nil {
		return out, fmt.Errorf("memory save outcome uncertain; no retry sent; refresh to verify")
	}
	return out, ValidateDshbotMemoryReceipt(cmd, out)
}
