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
	"whaleisle/internal/store/storelocale"
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
		Source string `json:"source"`
		Magpie *struct {
			File  string `json:"file"`
			Calls int64  `json:"calls"`
			From  *int64 `json:"from"`
			To    *int64 `json:"to"`
		} `json:"magpie"`
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
	generation                       uint64
	Range                            DateRange
	RangeStart, RangeEnd, RangeError string
	Raw                              json.RawMessage
	Loading                          bool
	Cached                           bool
	Err                              string
	Days                             int
	// UpdatedAt is the Host's `updatedAt` for this snapshot (epoch seconds),
	// 0 when the response carried none.
	UpdatedAt int64
}
type StatsView struct {
	raw                              json.RawMessage
	Range                            DateRange
	RangeStart, RangeEnd, RangeError string
	Overview                         *Overview
	Loading                          bool
	Cached                           bool
	Err                              string
	Days                             int
	// UpdatedAt is the Host's updatedAt for this snapshot (epoch seconds),
	// 0 when the response carried none.
	UpdatedAt int64
}

func (s *Store) Stats() StatsView {
	s.mu.Lock()
	raw := append(json.RawMessage(nil), s.stats.Raw...)
	v := StatsView{raw: raw, Range: s.stats.Range, RangeStart: s.stats.RangeStart, RangeEnd: s.stats.RangeEnd, RangeError: s.stats.RangeError, Loading: s.stats.Loading, Cached: s.stats.Cached, Err: s.stats.Err, Days: s.stats.Days}
	s.mu.Unlock()
	if v.Days == 0 {
		v.Days = 7
	}
	if len(raw) > 0 {
		var o Overview
		if json.Unmarshal(raw, &o) == nil {
			v.Overview = &o
			v.UpdatedAt = o.UpdatedAt
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
	s.clearRepairConfirmationLocked()
	s.stats.generation++
	generation := s.stats.generation
	s.stats.Loading = true
	s.stats.Err = ""
	ep := client.Endpoint
	cachePath := s.statsCachePath
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		httpClient := client.HTTP
		if httpClient == nil {
			httpClient = &http.Client{Timeout: 60 * time.Second}
		}
		rpc := &pluginrpc.Client{BaseURL: ep.Origin, Cookie: ep.Cookie, HTTPClient: httpClient}
		var raw json.RawMessage
		err := rpc.Call(ctx, "usage-stats", "overview", map[string]any{"force": force}, &raw)
		if err == nil {
			err = decodeOverview(raw)
		}
		s.mu.Lock()
		if epoch != s.statsEpoch || client != s.client || generation != s.stats.generation {
			s.mu.Unlock()
			return
		}
		s.stats.Loading = false
		s.statsCancel = nil
		var writeCache bool
		if err != nil {
			s.stats.Err = storelocale.T(storelocale.KeyStoreUsageUsageStatisticsUnavailable) + err.Error()
		} else {
			s.stats.Raw = append(json.RawMessage(nil), raw...)
			s.stats.Cached = false
			writeCache = cachePath != "" && s.statsCachePath == cachePath
			s.reconcileStatsRangeLocked()
		}
		s.finishRepairScanLocked(err)
		s.mu.Unlock()
		if writeCache {
			writeStatsCacheFile(cachePath, raw)
		}
		s.fire()
	}()
}

// ExportStats preserves the full host payload for JSON, including billing buckets.
// CSV keeps the installed daily/model headers. Heatmap selections are local
// summaries and never narrow the raw overview or these export contracts.
func (s *Store) ExportStats(format string) ([]byte, error) {
	v := s.Stats()
	billing := s.Billing()
	settings := billing.Settings
	if !billing.Loaded {
		settings = BillingSettings{PeakValleyEnabled: true}
	}
	raw := v.raw
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
	w.UseCRLF = false
	headers := []string{"total", "input", "output", "cacheRead", "cacheWrite", "costPeakCents", "costOffPeakCents", "costTotalCents"}
	switch format {
	case "days":
		w.Write(append([]string{"date"}, headers...))
	case "models":
		w.Write(append([]string{"model", "provider"}, headers...))
	case "sessions":
		w.Write(append([]string{"session", "provider"}, headers...))
	default:
		return nil, errors.New("unknown usage export format")
	}
	write := func(kind, name, provider string, t Totals, models []ModelUsage) {
		values := []string{name}
		if format != "days" {
			values = append(values, provider)
		}
		for _, n := range []float64{t.Input + t.Output, t.Input, t.Output, t.CacheRead, t.CacheWrite} {
			values = append(values, strconv.FormatFloat(n, 'f', -1, 64))
		}
		peak, idle, priced := 0.0, 0.0, 0
		for _, model := range models {
			p, i, ok := EstimateModelPhases(model, settings)
			if !ok {
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
			if len(d.Models) == 0 {
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
			if t.Input+t.Output+t.CacheRead+t.CacheWrite <= 0 {
				continue
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
