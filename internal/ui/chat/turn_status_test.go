package chat

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestFormatTurnStatus(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	if got := formatTurnStatus("0"); got != i18n.T(i18n.KeyChatTurnDone) {
		t.Fatalf("zero = %q", got)
	}
	if got := formatTurnStatus("45000"); got != "已完成，用时 45 秒" {
		t.Fatalf("sec = %q", got)
	}
	if got := formatTurnStatus("78000"); got != "已完成，用时 1 分 18 秒" {
		t.Fatalf("minsec = %q", got)
	}
	if got := formatTurnStatus("120000"); got != "已完成，用时 2 分" {
		t.Fatalf("min = %q", got)
	}
}
