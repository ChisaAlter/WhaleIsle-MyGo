package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestTaskProtectionErrorPreservesCuratedLocalizedCopy(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(i18n.DefaultLocale)

	store := session.New()
	s := &mainState{sessions: store}

	curated := i18n.T(i18n.KeyCmdRuntimeChangedRetryRestart)
	s.taskProtectionError(nil, localized(curated))
	if got := store.Snapshot().LastAction; got != curated {
		t.Fatalf("localized error wiped: got %q, want %q", got, curated)
	}

	// Wrapped localized errors must also preserve their exact curated sentence.
	custom := "导入仍在进行中，请等待完成。"
	s.taskProtectionError(nil, fmt.Errorf("commit rejected: %w", localized(custom)))
	if got := store.Snapshot().LastAction; got != custom {
		t.Fatalf("wrapped localized error wiped: got %q, want %q", got, custom)
	}

	// Non-localized raw errors must fall back to the generic catalog sentence
	// without leaking technical text.
	raw := errors.New(`open C:\secret\file: Access is denied.`)
	s.taskProtectionError(nil, raw)
	got := store.Snapshot().LastAction
	wantGeneric := i18n.T(i18n.KeyProtectedActionFailed)
	if got != wantGeneric {
		t.Fatalf("raw error copy = %q, want %q", got, wantGeneric)
	}
	if strings.Contains(got, "Access is denied") || strings.Contains(got, `C:\secret`) {
		t.Fatalf("raw error leaked into UI copy: %q", got)
	}

	// Nil and ErrCancelled are silent no-ops and keep the previous status.
	s.taskProtectionError(nil, nil)
	s.taskProtectionError(nil, taskcontrol.ErrCancelled)
	if after := store.Snapshot().LastAction; after != wantGeneric {
		t.Fatalf("cancelled/nil error mutated status line: %q", after)
	}
}
