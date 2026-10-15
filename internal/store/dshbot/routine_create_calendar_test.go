package dshbot

import (
	"errors"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func TestBotRoutineCalendarScheduleFields(t *testing.T) {
	for _, tc := range []struct{ frequency, weekday, monthDay, cron, want string }{
		{"hourly", "", "", "", "every 1h"},
		{"weekdays", "", "", "", "5 9 * * 1-5"},
		{"weekly", "0", "", "", "5 9 * * 0"},
		{"weekly", "6", "", "", "5 9 * * 6"},
		{"monthly", "", "1", "", "5 9 1 * *"},
		{"monthly", "", "31", "", "5 9 31 * *"},
		{"advanced", "", "", " 0  9 31 * * ", "0  9 31 * *"},
		{"advanced", "", "", "*/15 9-17 * * 1-5", "*/15 9-17 * * 1-5"},
		// Original client checks shape only; the Host must validate parser semantics.
		{"advanced", "", "", "61 * * * *", "61 * * * *"},
	} {
		d := RoutineCreateDraft{Frequency: tc.frequency, Hour: "09", Minute: "05", Weekday: tc.weekday, MonthDay: tc.monthDay, Cron: tc.cron}
		got, err := routineCreateSchedule(d)
		if err != nil || got != tc.want {
			t.Fatalf("%+v => %q %v", d, got, err)
		}
	}
	for _, d := range []RoutineCreateDraft{
		{Frequency: "weekly", Hour: "09", Minute: "00", Weekday: "7"},
		{Frequency: "weekly", Hour: "09", Minute: "00", Weekday: "-1"},
		{Frequency: "weekly", Hour: "09", Minute: "00", Weekday: "1.5"},
		{Frequency: "monthly", Hour: "09", Minute: "00", MonthDay: "0"},
		{Frequency: "monthly", Hour: "09", Minute: "00", MonthDay: "32"},
		{Frequency: "monthly", Hour: "09", Minute: "00", MonthDay: "2.5"},
		{Frequency: "weekdays", Hour: "24", Minute: "00"},
		{Frequency: "weekdays", Hour: "09", Minute: "60"},
		{Frequency: "advanced", Cron: "0 0 9 * * *"},
		{Frequency: "advanced", Cron: "0 9 * *"},
		{Frequency: "advanced", Cron: strings.Repeat("1", 41) + " * * * *"},
		{Frequency: "advanced", Cron: strings.Repeat("1", 40) + " " + strings.Repeat("1", 40) + " " + strings.Repeat("1", 40) + " " + strings.Repeat("1", 40) + " " + strings.Repeat("1", 40)},
		{Frequency: "advanced", Cron: "0 9 * * *\x00"},
	} {
		if got, err := routineCreateSchedule(d); err == nil {
			t.Errorf("invalid %+v => %q", d, got)
		}
	}
	for _, frequency := range []string{"once", "hourly", "daily", "weekdays", "weekly", "monthly", "interval", "advanced"} {
		if !routineCreateFrequencySupported(frequency) {
			t.Fatal(frequency)
		}
	}
	if routineCreateFrequencySupported("invented") {
		t.Fatal("invented frequency accepted")
	}
}

func TestBotRoutineCalendarReviewAndRejectedDraft(t *testing.T) {
	var c RoutineCreator
	k := RoutineScope{HostKey: "fixture-account", BotID: "b"}
	c.Sync(k, 1, RoutineSnapshot{Scope: RoutineScope{HostKey: "client", BotID: "b"}, Loaded: true, Revision: 4, SessionID: "s"})
	c.Start(k, "America/New_York")
	c.Change(k, "prompt", "Fixture prompt")
	d := c.Snapshot(k).Draft
	if d.Weekday != "1" || d.MonthDay != "1" || d.Cron != "0 9 * * *" {
		t.Fatal("original defaults missing", d)
	}
	c.Change(k, "frequency", "monthly")
	c.Change(k, "monthDay", "31")
	c.Change(k, "hour", "02")
	c.Change(k, "minute", "30")
	c.Prepare(k)
	monthly := c.Snapshot(k).Review
	if monthly == nil || monthly.Command.Edit.Schedule != "30 2 31 * *" || monthly.Command.Edit.Timezone != "America/New_York" {
		t.Fatal(monthly)
	}
	c.Change(k, "frequency", "advanced")
	c.Change(k, "cron", "61 * * * *")
	c.Prepare(k)
	if _, _, ok := c.Begin(k, monthly); ok {
		t.Fatal("old monthly review consumed cron draft")
	}
	review := c.Snapshot(k).Review
	ticket, cmd, ok := c.Begin(k, review)
	if !ok || !c.MarkSent(ticket) {
		t.Fatal("explicit cron review missing")
	}
	c.Finish(ticket, pluginrpc.DshbotRoutineCreateResult{Rejected: true}, errors.New("pluginrpc: failed: Invalid cron schedule: Constraint error"))
	s := c.Snapshot(k)
	if s.Uncertain || s.Draft.Cron != "61 * * * *" || s.Draft.RequestID != cmd.RequestID {
		t.Fatal("Host rejection lost draft or became ambiguous")
	}
	c.Change(k, "cron", "30 2 * * *")
	c.Prepare(k)
	if c.Snapshot(k).Review == nil || c.Snapshot(k).Review.Command.Edit.Schedule != "30 2 * * *" {
		t.Fatal("corrected cron not reviewable")
	}
}
