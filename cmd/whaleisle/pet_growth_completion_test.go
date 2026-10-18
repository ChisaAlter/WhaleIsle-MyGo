package main

import (
	"context"
	"errors"
	"testing"
)

func TestPetGrowthRefreshOnlyNeverConsumes(t *testing.T) {
	food, consumed := 3, 0
	finish := func(_ string, commit func() error) bool {
		if commit != nil {
			_ = commit()
		}
		return true
	}
	finishPetGrowthRefresh(context.Background(), nil, false, finish, func() error { food += 7; return nil }, func(bool) { consumed++; food-- })
	if food != 10 || consumed != 0 {
		t.Fatalf("automatic refresh changed care/consumption: %d %d", food, consumed)
	}
}
func TestPetGrowthExplicitRefreshThenSingleFeed(t *testing.T) {
	food, consumed := 0, 0
	active := true
	finish := func(_ string, commit func() error) bool {
		if !active {
			return false
		}
		active = false
		if commit != nil {
			_ = commit()
		}
		return true
	}
	refresh := func() error { food = 7; return nil }
	feed := func(fresh bool) {
		if !fresh {
			t.Error("successful scan marked stale")
		}
		consumed += food
		food = 0
	}
	finishPetGrowthRefresh(context.Background(), nil, true, finish, refresh, feed)
	finishPetGrowthRefresh(context.Background(), nil, true, finish, refresh, feed)
	if consumed != 7 || food != 0 {
		t.Fatalf("duplicate completion reconsumed: %d %d", food, consumed)
	}
}
func TestPetGrowthFailureAndCancellationKeepBalances(t *testing.T) {
	for _, kind := range []string{"cancelled", "closed", "save-failed"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancelled" {
				cancel()
			}
			food, consumed, refreshes := 7, 0, 0
			finish := func(_ string, commit func() error) bool {
				if kind == "closed" {
					return false
				}
				if commit != nil {
					_ = commit()
				}
				return true
			}
			finishPetGrowthRefresh(ctx, nil, true, finish, func() error { refreshes++; return errors.New("save failed") }, func(bool) { consumed++; food = 0 })
			if food != 7 || consumed != 0 {
				t.Fatalf("failed/stale operation consumed: %d %d", food, consumed)
			}
			if kind != "save-failed" && refreshes != 0 {
				t.Fatal("cancelled/closed operation refreshed")
			}
		})
	}
}
func TestPetGrowthFailedScanUsesOnlySavedFoodForExplicitFeed(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		food, consumed := 7, 0
		finish := func(_ string, commit func() error) bool {
			if commit != nil {
				t.Fatal("failed scan attempted settlement")
			}
			return true
		}
		finishPetGrowthRefresh(context.Background(), errors.New("scan unavailable"), explicit, finish, func() error { t.Fatal("failed scan refreshed"); return nil }, func(fresh bool) {
			if fresh {
				t.Fatal("fallback marked fresh")
			}
			consumed = food
			food = 0
		})
		if explicit && (food != 0 || consumed != 7) {
			t.Fatal("explicit fallback did not consume saved food")
		}
		if !explicit && (food != 7 || consumed != 0) {
			t.Fatal("automatic failure consumed saved food")
		}
	}
}
