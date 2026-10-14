package usage

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
)

// These shapes follow the built-in dsh-usage-panel shared/contract.ts, version 4.
// The host owns aggregation and inherited-session boundaries; jobs are never tokens.
type Totals struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}
type ModelUsage struct {
	Cost *PhaseBuckets `json:"cost"`
	Totals
	Model    string `json:"model"`
	Provider string `json:"provider"`
}
type DayUsage struct {
	Date   string            `json:"date"`
	Total  float64           `json:"total"`
	Models map[string]Totals `json:"models"`
}
type SessionUsage struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Totals Totals `json:"totals"`
	Depth  int    `json:"depth"`
}
type Overview struct {
	Days         []DayUsage   `json:"days"`
	Totals       Totals       `json:"totals"`
	SessionCount int          `json:"sessionCount"`
	ByModel      []ModelUsage `json:"byModel"`
	AllTime      struct {
		Totals       Totals       `json:"totals"`
		SessionCount int          `json:"sessionCount"`
		ByModel      []ModelUsage `json:"byModel"`
	} `json:"allTime"`
	Coverage struct {
		Mode             string   `json:"mode"`
		Timezone         string   `json:"timezone"`
		SessionsTotal    int      `json:"sessionsTotal"`
		SessionsOk       int      `json:"sessionsOk"`
		SessionsFailed   int      `json:"sessionsFailed"`
		SessionsPending  int      `json:"sessionsPending"`
		FailedSessionIDs []string `json:"failedSessionIds"`
	} `json:"coverage"`
	TopSessions []SessionUsage `json:"topSessions"`
	UpdatedAt   int64          `json:"updatedAt"`
	Stale       bool           `json:"stale"`
}
type statsState struct {
	Raw     json.RawMessage
	Loading bool
	Err     string
	Days    int
}
type StatsView struct {
	Overview *Overview
	Loading  bool
	Err      string
	Days     int
}

func (s *Store) Stats() StatsView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := StatsView{Loading: s.stats.Loading, Err: s.stats.Err, Days: s.stats.Days}
	if v.Days == 0 {
		v.Days = 7
	}
	if len(s.stats.Raw) > 0 {
		var o Overview
		if json.Unmarshal(s.stats.Raw, &o) == nil {
			v.Overview = &o
		}
	}
	return v
}
func (s *Store) SetStatsDays(days int) {
	if days != 7 && days != 14 && days != 30 {
		return
	}
	s.mu.Lock()
	s.stats.Days = days
	s.mu.Unlock()
	s.fire()
}
func decodeOverview(raw json.RawMessage) error {
	var o Overview
	if err := json.Unmarshal(raw, &o); err != nil {
		return err
	}
	if o.UpdatedAt <= 0 || o.Coverage.Timezone != "UTC" || o.Days == nil || o.AllTime.ByModel == nil || o.TopSessions == nil {
		return errors.New("usage-stats: incomplete overview contract")
	}
	return nil
}
func (s *Store) RefreshStats(ctx context.Context, force bool) {
	s.mu.Lock()
	client := s.client
	if client == nil || s.stats.Loading {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 65*time.Second)
	s.statsCancel = cancel
	epoch := s.statsEpoch
	s.stats.Loading = true
	s.stats.Err = ""
	ep := client.Endpoint
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		rpc := &pluginrpc.Client{BaseURL: ep.Origin, Cookie: ep.Cookie, HTTPClient: &http.Client{Timeout: 60 * time.Second}}
		var raw json.RawMessage
		err := rpc.Call(ctx, "usage-stats", "overview", map[string]any{"force": force}, &raw)
		if err == nil {
			err = decodeOverview(raw)
		}
		s.mu.Lock()
		if epoch != s.statsEpoch {
			s.mu.Unlock()
			return
		}
		s.stats.Loading = false
		s.statsCancel = nil
		if err != nil {
			s.stats.Err = "用量统计后端不可用 / Usage statistics unavailable: " + err.Error()
		} else {
			s.stats.Raw = append(json.RawMessage(nil), raw...)
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ExportStats preserves the full host payload for JSON, including billing buckets.
// CSV exports token rows only, with no invented estimates for unpriced models.
func (s *Store) ExportStats(format string) ([]byte, error) {
	s.mu.Lock()
	raw := append(json.RawMessage(nil), s.stats.Raw...)
	s.mu.Unlock()
	if len(raw) == 0 {
		return nil, errors.New("no usage overview to export")
	}
	if format == "json" {
		var out bytes.Buffer
		err := json.Indent(&out, raw, "", "  ")
		return out.Bytes(), err
	}
	var o Overview
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("\xef\xbb\xbf")
	w := csv.NewWriter(&b)
	w.UseCRLF = true
	w.Write([]string{"kind", "name", "provider", "total", "input", "output", "cacheRead", "cacheWrite"})
	write := func(kind, name, provider string, t Totals) {
		values := []string{kind, name, provider}
		for _, n := range []float64{t.Total, t.Input, t.Output, t.CacheRead, t.CacheWrite} {
			values = append(values, strconv.FormatFloat(n, 'f', -1, 64))
		}
		for i, v := range values {
			if strings.ContainsAny(strings.TrimLeft(v, " \t\r\n")[:min(1, len(strings.TrimLeft(v, " \t\r\n")))], "=+-@") {
				values[i] = "'" + v
			}
		}
		w.Write(values)
	}
	switch format {
	case "models":
		for _, m := range o.AllTime.ByModel {
			write("model", m.Model, m.Provider, m.Totals)
		}
	case "days":
		for _, d := range o.Days {
			t := Totals{Total: d.Total}
			for _, m := range d.Models {
				t.Input += m.Input
				t.Output += m.Output
				t.CacheRead += m.CacheRead
				t.CacheWrite += m.CacheWrite
			}
			write("day_UTC", d.Date, "", t)
		}
	case "sessions":
		for _, row := range o.TopSessions {
			write("top_session", fmt.Sprintf("%s (%s)", row.Title, row.ID), "", row.Totals)
		}
	default:
		return nil, errors.New("unknown usage export format")
	}
	w.Flush()
	return b.Bytes(), w.Error()
}
