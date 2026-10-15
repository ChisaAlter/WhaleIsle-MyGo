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
	Date       string                   `json:"date"`
	Total      float64                  `json:"total"`
	Models     map[string]Totals        `json:"models"`
	ModelCosts map[string]*PhaseBuckets `json:"modelCosts"`
}
type SessionUsage struct {
	Models     []ModelUsage `json:"models"`
	LastActive int64        `json:"lastActive"`
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	Totals     Totals       `json:"totals"`
	Depth      int          `json:"depth"`
}
type ProviderUsage struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Totals Totals `json:"totals"`
}
type Overview struct {
	Providers    []ProviderUsage `json:"providers"`
	Days         []DayUsage      `json:"days"`
	Totals       Totals          `json:"totals"`
	SessionCount int             `json:"sessionCount"`
	ByModel      []ModelUsage    `json:"byModel"`
	AllTime      struct {
		Totals       Totals       `json:"totals"`
		SessionCount int          `json:"sessionCount"`
		ByModel      []ModelUsage `json:"byModel"`
	} `json:"allTime"`
	Coverage struct {
		Mode                  string   `json:"mode"`
		Timezone              string   `json:"timezone"`
		SessionsTotal         int      `json:"sessionsTotal"`
		SessionsOk            int      `json:"sessionsOk"`
		SessionsFailed        int      `json:"sessionsFailed"`
		SessionsPending       int      `json:"sessionsPending"`
		FailedSessionIDs      []string `json:"failedSessionIds"`
		EventsCounted         int64    `json:"eventsCounted"`
		Retries               int64    `json:"retries"`
		CompactionTokens      float64  `json:"compactionTokens"`
		UsageSessionsMain     int      `json:"usageSessionsMain"`
		UsageSessionsSubagent int      `json:"usageSessionsSubagent"`
	} `json:"coverage"`
	TopSessions []SessionUsage `json:"topSessions"`
	UpdatedAt   int64          `json:"updatedAt"`
	Stale       bool           `json:"stale"`
}
type statsState struct {
	Range                            DateRange
	RangeStart, RangeEnd, RangeError string
	Raw                              json.RawMessage
	Loading                          bool
	Err                              string
	Days                             int
}
type StatsView struct {
	raw                              json.RawMessage
	Range                            DateRange
	RangeStart, RangeEnd, RangeError string
	Overview                         *Overview
	Loading                          bool
	Err                              string
	Days                             int
}

func (s *Store) Stats() StatsView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := StatsView{raw: append(json.RawMessage(nil), s.stats.Raw...), Range: s.stats.Range, RangeStart: s.stats.RangeStart, RangeEnd: s.stats.RangeEnd, RangeError: s.stats.RangeError, Loading: s.stats.Loading, Err: s.stats.Err, Days: s.stats.Days}
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
	if client == nil || s.stats.Loading || s.repair.Busy {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 65*time.Second)
	s.statsCancel = cancel
	epoch := s.statsEpoch
	s.resetRankingLocked()
	s.repair.PendingID = ""
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
			s.reconcileStatsRangeLocked()
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ExportStats preserves the full host payload for JSON, including billing buckets.
// CSV uses recent overview.byModel (or the selected UTC range), and reports
// CNY cents plus priced/unpriced counts so partial estimates cannot imply a bill.
func (s *Store) ExportStats(format string) ([]byte, error) {
	v := s.Stats()
	billing := s.Billing()
	settings := billing.Settings
	if !billing.Loaded {
		settings = BillingSettings{PeakValleyEnabled: true}
	}
	if v.Range.Active() && format == "sessions" {
		return nil, errors.New("session ranking is all-time only; clear the date selection to export sessions")
	}
	raw := v.raw
	if len(raw) == 0 {
		return nil, errors.New("no usage overview to export")
	}
	if v.Range.Active() {
		var err error
		raw, err = filteredExport(raw, v)
		if err != nil {
			return nil, err
		}
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
	w.Write([]string{"kind", "name", "provider", "total", "input", "output", "cacheRead", "cacheWrite", "costPeakCents", "costOffPeakCents", "costTotalCents", "currency", "pricedModels", "unpricedModels"})
	write := func(kind, name, provider string, t Totals, models []ModelUsage) {
		values := []string{kind, name, provider}
		for _, n := range []float64{t.Total, t.Input, t.Output, t.CacheRead, t.CacheWrite} {
			values = append(values, strconv.FormatFloat(n, 'f', -1, 64))
		}
		peak, idle, priced, unpriced := 0.0, 0.0, 0, 0
		for _, model := range models {
			p, i, ok := EstimateModelPhases(model, settings)
			if !ok {
				unpriced++
				continue
			}
			peak += p
			idle += i
			priced++
		}
		costs := []string{"", "", ""}
		if priced > 0 {
			for i, n := range []float64{peak, idle, peak + idle} {
				costs[i] = strconv.FormatFloat(n, 'f', 0, 64)
			}
		}
		values = append(values, costs...)
		values = append(values, "CNY", strconv.Itoa(priced), strconv.Itoa(unpriced))
		for i, v := range values {
			trimmed := strings.TrimLeft(v, " \t\r\n")
			if strings.ContainsAny(trimmed[:min(1, len(trimmed))], "=+-@") {
				values[i] = "'" + v
			}
		}
		w.Write(values)
	}
	switch format {
	case "models":
		for _, m := range o.ByModel {
			write("model", m.Model, m.Provider, m.Totals, []ModelUsage{m})
		}
	case "days":
		providers := map[string]string{}
		// Keep provider attribution even when filtered JSON omits all-time data.
		for _, m := range v.Overview.AllTime.ByModel {
			providers[m.Model] = m.Provider
		}
		for _, d := range o.Days {
			if d.Total <= 0 {
				continue
			}
			t := Totals{Total: d.Total}
			models := make([]ModelUsage, 0, len(d.Models))
			for name, m := range d.Models {
				t.Input += m.Input
				t.Output += m.Output
				t.CacheRead += m.CacheRead
				t.CacheWrite += m.CacheWrite
				models = append(models, ModelUsage{Model: name, Provider: providers[name], Totals: m, Cost: d.ModelCosts[name]})
			}
			write("day_UTC", d.Date, "", t, models)
		}
	case "sessions":
		for _, row := range o.TopSessions {
			write("top_session", fmt.Sprintf("%s (%s)", row.Title, row.ID), "", row.Totals, row.Models)
		}
	default:
		return nil, errors.New("unknown usage export format")
	}
	w.Flush()
	return b.Bytes(), w.Error()
}
