package petgrowth

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Usage keeps source identity and the gateway's absolute local-day count next
// to the settlements. The existing Scan API remains sessions-only.
type Usage struct {
	// CacheInclusive is migration evidence from the same decoded settlements;
	// it never contributes food, usage totals or daily counts.
	CacheInclusive map[string]int64
	Samples        map[string]int64
	Source         string
	TodayUsed      *int64
	Total          *int64
}

func MagpieUsageFile() string {
	if p := strings.TrimSpace(os.Getenv("MAGPIE_USAGE_FILE")); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "magpie", "usage.jsonl")
}

func ScanUsage(ctx context.Context, node, home, usageFile string) (Usage, error) {
	if usage, err := ReadMagpieUsage(usageFile, time.Now()); err == nil && usage != nil {
		return *usage, nil
	}
	return scanSessionUsage(ctx, node, home)
}

// ReadMagpieUsage consumes only dsh request metadata. A repeated
// route replaces that request's settlement; cache counters are not headline
// in/out tokens. Missing/unreadable/other-agent logs leave the session scanner
// authoritative, matching the original desktop's source selection.
func ReadMagpieUsage(file string, now time.Time) (*Usage, error) {
	if file == "" || !filepath.IsAbs(file) {
		return nil, os.ErrNotExist
	}
	stat, err := os.Lstat(file)
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsRegular() || stat.Size() > 512<<20 {
		return nil, errors.New("invalid gateway usage source")
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	type settled struct {
		amount int64
		day    string
	}
	rows := map[string]settled{}
	calls := 0
	var today, total int64
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for scanner.Scan() {
		var row map[string]any
		decoder := json.NewDecoder(strings.NewReader(scanner.Text()))
		decoder.UseNumber()
		if decoder.Decode(&row) != nil || row["agent"] != "dsh" {
			continue
		}
		count := func(v any) int64 {
			number, ok := v.(json.Number)
			if !ok {
				return 0
			}
			n, err := number.Float64()
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 || n > float64(MaxCount) {
				return 0
			}
			return int64(math.Floor(n))
		}
		calls++
		amount, err := add(count(row["in"]), count(row["out"]))
		if err != nil {
			return nil, err
		}
		total, err = add(total, amount)
		if err != nil {
			return nil, err
		}
		at, _ := row["t"].(string)
		if atTime, parseErr := time.Parse(time.RFC3339Nano, at); parseErr == nil && atTime.In(now.Location()).Format("2006-01-02") == now.Format("2006-01-02") {
			today, err = add(today, amount)
			if err != nil {
				return nil, err
			}
		}
		identity := ""
		for _, key := range []string{"route_id", "request_id", "rid", "response_id"} {
			if v := row[key]; v != nil && v != "" {
				identity = "route:" + fmt.Sprint(v)
				break
			}
		}
		if identity == "" && at != "" {
			part := func(key string) string {
				if v := row[key]; v != nil {
					return fmt.Sprint(v)
				}
				return ""
			}
			identity = "row:" + at + "|" + part("session") + "|" + part("status") + "|" + part("model")
		}
		if identity == "" || amount == 0 {
			continue
		}
		raw, _ := json.Marshal([]string{"magpie", identity})
		sum := sha256.Sum256(raw)
		id := hex.EncodeToString(sum[:])
		day := ""
		if atTime, err := time.Parse(time.RFC3339Nano, at); err == nil {
			day = atTime.In(now.Location()).Format("2006-01-02")
		}
		rows[id] = settled{amount, day}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if calls == 0 {
		return nil, os.ErrNotExist
	}
	samples := make(map[string]int64, len(rows))
	for id, row := range rows {
		samples[id] = row.amount
	}
	if _, err := ValidateSamples(samples); err != nil {
		return nil, err
	}
	return &Usage{Samples: samples, Source: "magpie", TodayUsed: &today, Total: &total}, nil
}
