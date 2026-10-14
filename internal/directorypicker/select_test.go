package directorypicker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Mock native picker and registrar; no dialogs, Host or real projects.
func TestSelectAndRegisterBoundary(t *testing.T) {
	chosen := t.TempDir()
	calls := 0
	register := func(_ context.Context, p string) error {
		calls++
		if p != chosen {
			t.Fatalf("registered wrong path %q", p)
		}
		return nil
	}
	got, err := SelectAndRegister(context.Background(), func() ([]string, error) { return []string{chosen}, nil }, register)
	if err != nil || got != chosen || calls != 1 {
		t.Fatalf("selection: %q %v calls=%d", got, err, calls)
	}
	got, err = SelectAndRegister(context.Background(), func() ([]string, error) { return nil, nil }, register)
	if !errors.Is(err, ErrCancelled) || got != "" || calls != 1 {
		t.Fatal("cancel registered or selected fallback")
	}
	ctx, cancel := context.WithCancel(context.Background())
	_, err = SelectAndRegister(ctx, func() ([]string, error) { cancel(); return []string{chosen}, nil }, register)
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("interrupted dialog registered path")
	}
	metadata := filepath.Join(chosen, ".git")
	if err = os.Mkdir(metadata, 0700); err != nil {
		t.Fatal(err)
	}
	_, err = SelectAndRegister(context.Background(), func() ([]string, error) { return []string{metadata}, nil }, register)
	if err == nil || calls != 1 {
		t.Fatal("Git metadata was registered")
	}
	_, err = SelectAndRegister(context.Background(), func() ([]string, error) { return []string{chosen}, nil }, func(context.Context, string) error { return errors.New("registration denied") })
	if err == nil {
		t.Fatal("registration failure ignored")
	}
}
