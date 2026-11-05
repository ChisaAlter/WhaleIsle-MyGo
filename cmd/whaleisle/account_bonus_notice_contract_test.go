package main

import (
	"testing"
	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
)

func TestPublishedResultBonusNoticeTitle(t *testing.T) {
	for _, title := range []string{"服务端奖励标题", ""} {
		if got := accountBonusNoticeTitle(usagestore.Bonus{Title: &title}); got != title {
			t.Fatalf("server title changed: %q != %q", got, title)
		}
	}
	if got := accountBonusNoticeTitle(usagestore.Bonus{}); got != i18n.T(i18n.KeyCmdRewardNotice) {
		t.Fatal("missing title lost existing localized fallback")
	}
}
