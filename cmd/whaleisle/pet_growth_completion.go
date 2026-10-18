package main

import "context"

// finishPetGrowthRefresh serializes settlement before an explicitly requested
// meal. finish owns the live-window gate; rejected, cancelled or failed saves
// never consume. As upstream, a scan error can still feed already-settled food.
func finishPetGrowthRefresh(ctx context.Context, scanErr error, wantsFeed bool,
	finish func(string, func() error) bool, refresh func() error, consume func(bool)) {
	if ctx.Err() != nil {
		finish("用量读取已取消，账本保持不变 / Cancelled", nil)
		return
	}
	if scanErr != nil {
		if finish("刷新未完成，保留已结算粮 / Scan unavailable; saved food retained", nil) && wantsFeed {
			consume(false)
		}
		return
	}
	saved := false
	accepted := finish("粮碗已刷新 / Local food refreshed", func() error {
		err := refresh()
		saved = err == nil
		return err
	})
	if accepted && saved && wantsFeed {
		consume(true)
	}
}
