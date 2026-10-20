package main

import (
	"context"
	"whaleisle/internal/ui/i18n"
)

// finishPetGrowthRefresh serializes settlement before an explicitly requested
// meal. finish owns the live-window gate; rejected, cancelled or failed saves
// never consume. As upstream, a scan error can still feed already-settled food.
func finishPetGrowthRefresh(ctx context.Context, scanErr error, wantsFeed bool,
	finish func(string, func() error) bool, refresh func() error, consume func(bool)) {
	if ctx.Err() != nil {
		finish(i18n.T(i18n.KeyCmdCancelled), nil)
		return
	}
	if scanErr != nil {
		if finish(i18n.T(i18n.KeyCmdScanUnavailableSavedFoodRetained), nil) && wantsFeed {
			consume(false)
		}
		return
	}
	saved := false
	accepted := finish(i18n.T(i18n.KeyCmdLocalFoodRefreshed), func() error {
		err := refresh()
		saved = err == nil
		return err
	})
	if accepted && saved && wantsFeed {
		consume(true)
	}
}
