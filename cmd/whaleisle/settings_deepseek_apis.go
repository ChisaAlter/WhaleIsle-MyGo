package main

import (
	"context"
	"strconv"
	"time"

	mygo "github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

const deepSeekAPITimeout = 45 * time.Second

// refreshDeepSeekAPIs re-reads the Host description and the credential
// reference's stored state. Called when the page opens, after a Host
// replacement, and when a person presses 重新加载.
// Both store methods run on their own async goroutines with internal timeouts,
// so callers must not attach a short-lived defer cancel() around the calls.
func (s *mainState) refreshDeepSeekAPIs() {
	store := s.settings
	if store == nil {
		return
	}
	if store.Snapshot().Value.Described {
		store.RefreshDescribeBackground(context.Background())
	} else {
		store.RefreshDescribe(context.Background())
	}
	ref := store.SnapshotDeepSeekTools().APIKeyEnv
	if ref == "" {
		ref = settingsstore.CredRefAPIKey
	}
	store.RefreshDeepSeekKey(context.Background(), ref)
	s.updateDeepSeekAPIsUI()
}

func (s *mainState) runDeepSeekAPIsSync(action func(context.Context)) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), deepSeekAPITimeout)
		defer cancel()
		action(ctx)
		s.updateDeepSeekAPIsUI()
	}()
}

func (s *mainState) updateDeepSeekAPIsUI() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) settingsDeepSeekAPIsAccess() *settings.DeepSeekToolsAccess {
	store := s.settings
	if store == nil {
		return nil
	}
	binding := func() *settingsstore.NamespaceBinding { return store.DeepSeekToolsBinding() }
	return &settings.DeepSeekToolsAccess{
		Snapshot: store.SnapshotDeepSeekTools(),
		Key:      store.SnapshotDeepSeekKey(),
		OnStage: func(field, value string) {
			if b := binding(); b != nil {
				b.SetDraft(field, value)
			}
		},
		// A switch writes the profile namespace right away, so the Host drops or
		// mounts that tool without a restart; the page follows on the next paint.
		OnToggle: func(field string, value bool) {
			b := binding()
			if b == nil {
				return
			}
			b.SetDraft(field, strconv.FormatBool(value))
			b.Save(context.Background())
			s.updateDeepSeekAPIsUI()
		},
		OnSave: func() {
			if b := binding(); b != nil {
				b.Save(context.Background())
				s.updateDeepSeekAPIsUI()
			}
		},
		OnReload: func() {
			if b := binding(); b != nil {
				b.Discard()
			}
			s.refreshDeepSeekAPIs()
		},
		OnRefresh: s.refreshDeepSeekAPIs,
		OnSaveKey: func(ref, value string) {
			s.runDeepSeekAPIsSync(func(ctx context.Context) { store.SaveDeepSeekKey(ctx, ref, value) })
		},
	}
}

func (s *mainState) renderDeepSeekAPIsSection(c *ui.Context) {
	access := s.settingsDeepSeekAPIsAccess()
	dark := mygo.Theme.IsDark()
	colors := theme.ResolveWithSystem(theme.IDSystem, dark).Active(dark)
	settings.DeepSeekToolsPane(c, access, theme.Color(colors.FG), theme.Color(colors.Muted))
}
