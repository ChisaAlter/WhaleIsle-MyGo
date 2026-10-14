package chat

import (
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestFormatUsageLabelEmpty(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	if got := FormatUsageLabel(nil, ""); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := FormatUsageLabel(nil, "  "); got != "" {
		t.Fatalf("blank balance = %q", got)
	}
	if got := FormatUsageLabel(nil, "(empty balance response)"); got != "" {
		t.Fatalf("placeholder = %q", got)
	}
	if got := FormatUsageLabel(nil, "(account/getBalance: timeout)"); got != "" {
		t.Fatalf("error stub = %q", got)
	}
}

func TestFormatUsageLabelJobsAndBalance(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	n := 3
	got := FormatUsageLabel(&n, `{"credits":12}`)
	if !strings.Contains(got, i18n.T(i18n.KeyChatUsageJobsToday)) {
		t.Fatalf("missing jobs key: %q", got)
	}
	if !strings.Contains(got, "3") {
		t.Fatalf("missing count: %q", got)
	}
	if !strings.Contains(got, i18n.T(i18n.KeyChatUsageBalance)) {
		t.Fatalf("missing balance key: %q", got)
	}
	if !strings.Contains(got, "credits") {
		t.Fatalf("missing balance stub: %q", got)
	}
	if !strings.Contains(got, " · ") {
		t.Fatalf("missing separator: %q", got)
	}
}

func TestFormatUsageLabelJobsZero(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	n := 0
	got := FormatUsageLabel(&n, "")
	wantPrefix := i18n.T(i18n.KeyChatUsageJobsToday) + " 0"
	if got != wantPrefix {
		t.Fatalf("got %q want %q", got, wantPrefix)
	}
}

func TestFormatUsageLabelBalanceOnly(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	got := FormatUsageLabel(nil, "ok")
	want := i18n.T(i18n.KeyChatUsageBalance) + " ok"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestCompactUsageBalanceTruncates(t *testing.T) {
	long := strings.Repeat("x", usageBalanceMaxRunes+10)
	got := CompactUsageBalance(long)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected ellipsis: %q", got)
	}
	if strings.Count(got, "x") != usageBalanceMaxRunes {
		t.Fatalf("rune count wrong: %q len=%d", got, strings.Count(got, "x"))
	}
}
