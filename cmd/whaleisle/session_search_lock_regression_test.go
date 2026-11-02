package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The full mainView owns native services that a headless test must not start.
// Check its actual writeback block, rather than reproducing its lock ordering in
// a test helper. The navigation-shell writeback below is also exercised live.
func TestSessionSearchRenderWritebackPublishesOutsideStateLock(t *testing.T) {
	for _, name := range []string{"main.go", "main_navigation.go"} {
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := checkSessionSearchWritebackLock(source); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for _, tc := range []struct {
		name, body string
		wantError  bool
	}{
		{"publish-after-unlock", `state.mu.Lock(); state.sessionSearch = query; state.mu.Unlock(); if sessions != nil { sessions.UpdateSessionSearch(ctx, query) }`, false},
		{"original-deadlock", `state.mu.Lock(); state.sessionSearch = query; if sessions != nil { sessions.UpdateSessionSearch(ctx, query) }; state.mu.Unlock()`, true},
		{"async-publish-loses-order", `state.mu.Lock(); state.sessionSearch = query; state.mu.Unlock(); go sessions.UpdateSessionSearch(ctx, query)`, true},
		{"writeback-outside-lock", `state.sessionSearch = query; sessions.UpdateSessionSearch(ctx, query)`, true},
		{"missing-publication", `state.mu.Lock(); state.sessionSearch = query; state.mu.Unlock()`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package fixture; func render() { " + tc.body + " }")
			if err := checkSessionSearchWritebackLock(source); (err != nil) != tc.wantError {
				t.Fatalf("constraint err=%v wantError=%v", err, tc.wantError)
			}
		})
	}
}

func checkSessionSearchWritebackLock(source []byte) error {
	file, err := parser.ParseFile(token.NewFileSet(), "render.go", source, 0)
	if err != nil {
		return err
	}
	selector := func(expr ast.Expr) string {
		var path func(ast.Expr) string
		path = func(expr ast.Expr) string {
			switch e := expr.(type) {
			case *ast.Ident:
				return e.Name
			case *ast.SelectorExpr:
				return path(e.X) + "." + e.Sel.Name
			}
			return ""
		}
		return path(expr)
	}
	blocks, problem := 0, error(nil)
	ast.Inspect(file, func(node ast.Node) bool {
		block, ok := node.(*ast.BlockStmt)
		if !ok || problem != nil {
			return problem == nil
		}
		owner := ""
		for _, stmt := range block.List {
			if assign, ok := stmt.(*ast.AssignStmt); ok {
				for _, lhs := range assign.Lhs {
					if field, ok := lhs.(*ast.SelectorExpr); ok && field.Sel.Name == "sessionSearch" {
						owner = selector(field.X)
					}
				}
			}
		}
		if owner == "" {
			return true
		}
		blocks++
		locked, wrote, published := false, false, 0
		for _, stmt := range block.List {
			ast.Inspect(stmt, func(child ast.Node) bool {
				if child == nil || problem != nil {
					return false
				}
				if _, ok := child.(*ast.FuncLit); ok {
					return false // A nested closure has a separate execution boundary.
				}
				var asynchronous ast.Node
				switch n := child.(type) {
				case *ast.GoStmt:
					asynchronous = n.Call
				case *ast.DeferStmt:
					asynchronous = n.Call
				}
				if asynchronous != nil {
					ast.Inspect(asynchronous, func(n ast.Node) bool {
						if call, ok := n.(*ast.CallExpr); ok && selector(call.Fun) == "sessions.UpdateSessionSearch" {
							problem = fmt.Errorf("search publication must remain synchronous after render writeback")
						}
						return true
					})
				}
				if assign, ok := child.(*ast.AssignStmt); ok {
					for _, lhs := range assign.Lhs {
						if selector(lhs) == owner+".sessionSearch" {
							if !locked {
								problem = fmt.Errorf("search state writeback is outside %s.mu", owner)
							}
							wrote = true
						}
					}
				}
				if call, ok := child.(*ast.CallExpr); ok {
					switch selector(call.Fun) {
					case owner + ".mu.Lock":
						locked = true
					case owner + ".mu.Unlock":
						locked = false
					case "sessions.UpdateSessionSearch":
						published++
						if locked || !wrote {
							problem = fmt.Errorf("search notify can reenter %s.mu before writeback unlock", owner)
						}
					}
				}
				return true
			})
		}
		if problem == nil && (published != 1 || locked) {
			problem = fmt.Errorf("expected one synchronous search publication after unlocking, got publications=%d locked=%v", published, locked)
		}
		return true
	})
	if problem != nil {
		return problem
	}
	if blocks == 0 {
		return fmt.Errorf("render search writeback block was not found")
	}
	return nil
}

func TestSessionSearchNavigationFrameNotifyReentersCommittedState(t *testing.T) {
	previousLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previousLocale) })
	store := session.New()
	s := &mainState{sessions: store}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var observedMu sync.Mutex
	var observed []string
	store.Attach(nil, func() {
		// Fail immediately rather than leaving a deadlocked tester goroutine.
		if !s.mu.TryLock() {
			t.Error("synchronous search notification reentered a locked mainState")
			return
		}
		committed := s.sessionSearch
		s.mu.Unlock()
		s.cancelComposerVoiceIfStale() // The real startup refresh callback's first action.
		observedMu.Lock()
		observed = append(observed, committed)
		observedMu.Unlock()
	})
	// Cancel any pending debounce before the test exits. No Host is attached.
	t.Cleanup(func() { store.UpdateSessionSearch(ctx, "") })
	sizes := frame.DefaultLayout()
	query, renameID, renameDraft := "", "", ""
	tt := ui.NewTester(func(c *ui.Context) {
		s.withMainNavigation(c, true, frame.Model{
			Store: store, Sizes: &sizes, ThemeID: theme.IDDeepseekLight,
			SessionSearch: &query, RenameSessionID: &renameID, RenameDraft: &renameDraft,
		}, func() { ui.Text(c, "Synthetic preserved conversation") })
	}, 1200, 800)
	if err := tt.Click(i18n.T(i18n.KeySidebarSearch)); err != nil {
		t.Fatal(err)
	}
	tt.Type("needle")
	assertLast := func(want string) int {
		t.Helper()
		observedMu.Lock()
		defer observedMu.Unlock()
		if len(observed) == 0 || observed[len(observed)-1] != want {
			t.Fatalf("notify did not synchronously observe committed query %q: %q", want, observed)
		}
		return len(observed)
	}
	count := assertLast("needle")
	for range 3 {
		tt.Frame()
	}
	if got := assertLast("needle"); got != count {
		t.Fatalf("unchanged render republished search: before=%d after=%d", count, got)
	}
	if err := tt.Click("Clear search"); err != nil {
		t.Fatal(err)
	}
	assertLast("")
	s.mu.Lock()
	committed := s.sessionSearch
	s.mu.Unlock()
	if committed != "" || query != "" {
		t.Fatalf("clear did not commit to state and input: %q/%q", committed, query)
	}
}
