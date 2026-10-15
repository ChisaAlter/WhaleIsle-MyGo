package dshbot

import (
	"errors"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func TestBotRoutineDurationScheduleGrammar(t *testing.T) {
	valid := []struct {
		draft RoutineCreateDraft
		want  string
	}{
		{RoutineCreateDraft{Frequency: "daily", Hour: "09", Minute: "05"}, "5 9 * * *"},
		{RoutineCreateDraft{Frequency: "once", OnceDelay: "30m"}, "30m"},
		{RoutineCreateDraft{Frequency: "once", OnceDelay: " 001H "}, "1h"},
		{RoutineCreateDraft{Frequency: "once", OnceDelay: "1m"}, "1m"},
		{RoutineCreateDraft{Frequency: "once", OnceDelay: "365d"}, "365d"},
		{RoutineCreateDraft{Frequency: "once", OnceDelay: "525600m"}, "525600m"},
		{RoutineCreateDraft{Frequency: "interval", IntervalAmount: "1", IntervalUnit: "h"}, "every 1h"},
		{RoutineCreateDraft{Frequency: "interval", IntervalAmount: " 001 ", IntervalUnit: "m"}, "every 1m"},
		{RoutineCreateDraft{Frequency: "interval", IntervalAmount: "8760", IntervalUnit: "h"}, "every 8760h"},
		{RoutineCreateDraft{Frequency: "interval", IntervalAmount: "365", IntervalUnit: "d"}, "every 365d"},
	}
	for _, tc := range valid {
		got, err := routineCreateSchedule(tc.draft)
		if err != nil || got != tc.want {
			t.Fatalf("%+v: %q, %v", tc.draft, got, err)
		}
	}
	for _, raw := range []string{"", "0m", "-1m", "+1m", "1.5h", "1e2m", "1s", "1w", "1 h", "every 1m", "366d", "8761h", "525601m", strings.Repeat("9", 201) + "m", "18446744073709551616m", "1m\x00"} {
		if got, err := routineCreateSchedule(RoutineCreateDraft{Frequency: "once", OnceDelay: raw}); err == nil {
			t.Errorf("invalid once accepted %q => %q", raw, got)
		}
	}
	for _, tc := range []struct{ amount, unit string }{{"0", "m"}, {"1.0", "h"}, {"1e2", "m"}, {"+1", "h"}, {"1", "H"}, {"1", "s"}, {"1", ""}, {"366", "d"}, {"525601", "m"}, {"18446744073709551616", "h"}} {
		if got, err := routineCreateSchedule(RoutineCreateDraft{Frequency: "interval", IntervalAmount: tc.amount, IntervalUnit: tc.unit}); err == nil {
			t.Errorf("invalid interval accepted %+v => %q", tc, got)
		}
	}
	if _, err := routineCreateSchedule(RoutineCreateDraft{Frequency: "advanced", Hour: "09", Minute: "00"}); err == nil {
		t.Fatal("unsupported type accepted")
	}
}

func TestBotRoutineDurationReviewIdentityAndPending(t *testing.T) {
	for _, frequency := range []string{"once", "interval"} {
		t.Run(frequency, func(t *testing.T) {
			var c RoutineCreator
			k := RoutineScope{HostKey: "fixture-account", BotID: "b"}
			source := RoutineSnapshot{Scope: RoutineScope{HostKey: "fixture-client", BotID: "b"}, Loaded: true, Revision: 7, SessionID: "s"}
			c.Sync(k, 1, source)
			c.Start(k, "Asia/Shanghai")
			d := c.Snapshot(k).Draft
			if d.Frequency != "daily" || d.OnceDelay != "30m" || d.IntervalAmount != "1" || d.IntervalUnit != "h" || d.Enabled {
				t.Fatal(d)
			}
			c.Change(k, "prompt", "Fixture prompt")
			c.Prepare(k)
			dailyReview := c.Snapshot(k).Review
			c.Change(k, "frequency", frequency)
			if c.Snapshot(k).Review != nil {
				t.Fatal("type change retained confirmation")
			}
			c.Change(k, "enabled", "true")
			c.Prepare(k)
			if _, _, ok := c.Begin(k, dailyReview); ok {
				t.Fatal("daily callback consumed duration review")
			}
			review := c.Snapshot(k).Review
			if review == nil {
				t.Fatal(c.Snapshot(k).Error)
			}
			want := "30m"
			if frequency == "interval" {
				want = "every 1h"
			}
			if review.Command.Edit.Schedule != want || review.Command.Edit.Timezone != "Asia/Shanghai" || !review.Command.Edit.Enabled || review.Command.Revision != 7 || review.Command.SessionID != "s" {
				t.Fatal(review)
			}
			ticket, cmd, ok := c.Begin(k, review)
			if !ok || !c.MarkSent(ticket) {
				t.Fatal("reviewed request missing")
			}
			c.Finish(ticket, pluginrpc.DshbotRoutineCreateResult{}, errors.New("synthetic lost response"))
			c.Change(k, "frequency", "daily")
			c.Change(k, "onceDelay", "1m")
			c.Change(k, "intervalAmount", "2")
			c.Prepare(k)
			s := c.Snapshot(k)
			if !s.Uncertain || s.Draft.Frequency != frequency || s.Draft.OnceDelay != "30m" || s.Draft.IntervalAmount != "1" || s.Draft.RequestID != cmd.RequestID {
				t.Fatal("pending duration identity changed")
			}
			c.Sync(k, 1, source)
			c.Rebase(k)
			if !c.Snapshot(k).Uncertain {
				t.Fatal("same revision replay unlocked")
			}
			source.Revision = 8
			c.Sync(k, 1, source)
			if !c.Snapshot(k).RetryVerified {
				t.Fatal("new revision fence not available")
			}
			c.Rebase(k)
			c.Prepare(k)
			s = c.Snapshot(k)
			if s.Review == nil || s.Review.Command.Edit.Schedule != want || s.Review.Command.RequestID == cmd.RequestID || s.Draft.Frequency != frequency {
				t.Fatal("rebase lost duration or reused request identity")
			}
		})
	}
}

func TestBotRoutineDurationInvalidDraftRetained(t *testing.T) {
	var c RoutineCreator
	k := RoutineScope{HostKey: "fixture-account", BotID: "b"}
	c.Sync(k, 1, RoutineSnapshot{Scope: RoutineScope{HostKey: "client", BotID: "b"}, Loaded: true, Revision: 1, SessionID: "s"})
	c.Start(k, "UTC")
	c.Change(k, "prompt", "Fixture prompt")
	c.Change(k, "frequency", "once")
	c.Change(k, "onceDelay", "0m")
	c.Prepare(k)
	s := c.Snapshot(k)
	if s.Review != nil || s.Error == "" || s.Draft.OnceDelay != "0m" {
		t.Fatal("invalid draft not retained")
	}
	c.Change(k, "onceDelay", "30m")
	c.Change(k, "timezone", "Local")
	c.Prepare(k)
	if c.Snapshot(k).Review != nil {
		t.Fatal("duration bypassed explicit timezone")
	}
	c.Change(k, "timezone", "UTC")
	c.Prepare(k)
	if c.Snapshot(k).Review == nil {
		t.Fatal(c.Snapshot(k).Error)
	}
	c.Change(k, "frequency", "advanced")
	if c.Snapshot(k).Draft.Frequency != "once" {
		t.Fatal("unsupported editor type accepted")
	}
	c.Change(k, "frequency", "interval")
	c.Change(k, "intervalUnit", "s")
	if c.Snapshot(k).Draft.IntervalUnit != "h" {
		t.Fatal("unsupported unit accepted")
	}
}
