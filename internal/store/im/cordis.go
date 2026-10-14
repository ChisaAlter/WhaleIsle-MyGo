package im

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh/api"
)

func probeCordisStatus(ctx context.Context, plugin *pluginrpc.Client) (rows []Entry, preview string, hits []string, lastErr error) {
	if plugin == nil {
		return nil, "", nil, nil
	}
	var parts []string
	for _, ch := range api.IMCordisChannels() {
		raw, err := plugin.IMChannelStatus(ctx, ch)
		if err != nil {
			lastErr = err
			continue
		}
		bots := parseCordisBots(raw, ch)
		if len(bots) == 0 {
			hits = append(hits, ch+" (empty roster)")
			continue
		}
		hits = append(hits, fmt.Sprintf("%s %d bot(s)", ch, len(bots)))
		rows = append(rows, bots...)
		parts = append(parts, fmt.Sprintf("%s: %d bots", ch, len(bots)))
	}
	if len(parts) > 0 {
		preview = strings.Join(parts, "\n")
		if len(preview) > 800 {
			preview = preview[:800] + "…"
		}
	}
	return rows, preview, hits, lastErr
}

func parseCordisBots(raw json.RawMessage, channel string) []Entry {
	var wrap struct {
		Snapshot           *cordisSnapshot `json:"snapshot"`
		Bots               []cordisBot     `json:"bots"`
		AgentPresetCatalog presetCatalog   `json:"agentPresetCatalog"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil
	}
	bots := wrap.Bots
	catalog := wrap.AgentPresetCatalog
	if wrap.Snapshot != nil {
		catalog = wrap.Snapshot.AgentPresetCatalog
		bots = wrap.Snapshot.Bots
	}
	out := make([]Entry, 0, len(bots))
	for _, b := range bots {
		id := strings.TrimSpace(b.BotID)
		if id == "" {
			continue
		}
		label := strings.TrimSpace(b.Bot.Name)
		if label == "" {
			label = channel + " " + id
		}
		out = append(out, Entry{
			ID:      channel + ":" + id,
			Label:   label,
			Detail:  cordisBotDetail(channel, b),
			Channel: channel, BotID: id, State: botState(b), Workspace: b.Workspace, AgentPreset: b.AgentPreset, Presets: catalog.presets(),
		})
	}
	return out
}

func cordisBotDetail(channel string, b cordisBot) string {
	state := botState(b)
	if b.Connected {
		state = "connected"
	}
	if state == "" {
		state = "offline"
	}
	parts := []string{channel, state}
	if summary := strings.TrimSpace(b.Health.Summary); summary != "" {
		parts = append(parts, summary)
	}
	if msg := strings.TrimSpace(b.LastMessageError.Message); msg != "" {
		if len(msg) > 120 {
			msg = msg[:120] + "…"
		}
		parts = append(parts, "lastMsg: "+msg)
	} else if b.Error != nil {
		if msg := strings.TrimSpace(b.Error.Message); msg != "" {
			if len(msg) > 120 {
				msg = msg[:120] + "…"
			}
			parts = append(parts, msg)
		}
	}
	return strings.Join(parts, " · ")
}

type cordisSnapshot struct {
	Bots               []cordisBot   `json:"bots"`
	AgentPresetCatalog presetCatalog `json:"agentPresetCatalog"`
}

type cordisBot struct {
	BotID       string `json:"botId"`
	Phase       string `json:"phase"`
	Workspace   string `json:"workspace"`
	AgentPreset string `json:"agentPreset"`
	Connected   bool   `json:"connected"`
	State       string `json:"state"`
	Health      struct {
		Summary string `json:"summary"`
	} `json:"health"`
	LastMessageError struct {
		Message string `json:"message"`
	} `json:"lastMessageError"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Bot struct {
		Name string `json:"name"`
	} `json:"bot"`
}

func botState(b cordisBot) string {
	if b.Connected {
		return "connected"
	}
	if b.State != "" {
		return b.State
	}
	if b.Phase != "" {
		return b.Phase
	}
	return "offline"
}
