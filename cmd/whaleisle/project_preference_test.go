package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/desktop/projectpreference"
)

func TestProjectMountPolicyMatchesInstalledGates(t *testing.T) {
	for _, item := range []struct {
		cfg  config.Config
		env  string
		want bool
	}{
		{config.Defaults(), "", false}, {config.Config{ProjectEnabled: true}, "", true}, {config.Config{}, "1", true}, {config.Config{}, "true", false},
		{config.Config{ProjectEnabled: true, SkipUserPlugins: true}, "1", false}, {config.Config{ProjectEnabled: true, DisabledPlugins: []string{"dsh-project"}}, "1", false},
	} {
		if got := projectMountEnabled(item.cfg, item.env); got != item.want {
			t.Fatalf("mount%+v env%q=%v want%v", item.cfg, item.env, got, item.want)
		}
	}
}

func TestProjectLifecycleTemporaryConfigMaintenanceAndRollback(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "ready", true: "restart failed"}[failed], func(t *testing.T) {
			root := t.TempDir()
			store, err := config.Open(filepath.Join(root, "settings"))
			if err != nil {
				t.Fatal(err)
			}
			if err = store.Update(func(c *config.Config) { c.Workspace = "owned"; c.Port = 4001; c.ProjectEnabled = true }); err != nil {
				t.Fatal(err)
			}
			before := store.Get()
			marker := filepath.Join(root, "project-data.keep")
			if err = os.WriteFile(marker, []byte("retained-project"), 0600); err != nil {
				t.Fatal(err)
			}
			c := new(projectpreference.Controller)
			restarts := 0
			err = c.Set(context.Background(), false, projectpreference.Transition{
				Read: func() bool { return store.Get().ProjectEnabled }, Write: func(expected, value bool) error {
					return store.UpdateChecked(func(cfg *config.Config) error {
						if cfg.ProjectEnabled != expected {
							return errors.New("conflict")
						}
						cfg.ProjectEnabled = value
						return nil
					})
				},
				Acquire:   func() (func(), error) { return importer.AcquireExternalMaintenance(filepath.Join(root, "home")) },
				LeaveLead: func(context.Context) error { return nil },
				Protect: func(_ context.Context, commit func() error) (bool, error) {
					if release, e := importer.AcquireExternalMaintenance(filepath.Join(root, "home")); e == nil {
						release()
						t.Fatal("lease not held through confirmation")
					}
					err := commit()
					return err == nil, err
				},
				Restart: func(context.Context) error {
					restarts++
					if store.Get().ProjectEnabled {
						t.Fatal("restart saw stale Project choice")
					}
					if failed {
						return errors.New("restart failed")
					}
					return nil
				},
			})
			if failed && err == nil || !failed && err != nil || restarts != 1 {
				t.Fatalf("restart=%d err=%v", restarts, err)
			}
			reopened, e := config.Open(filepath.Join(root, "settings"))
			if e != nil {
				t.Fatal(e)
			}
			want := before
			want.ProjectEnabled = failed
			if !reflect.DeepEqual(reopened.Get(), want) {
				t.Fatal("lifecycle changed unrelated config or failed rollback")
			}
			if data, e := os.ReadFile(marker); e != nil || string(data) != "retained-project" {
				t.Fatal("Project data was removed")
			}
			release, e := importer.AcquireExternalMaintenance(filepath.Join(root, "home"))
			if e != nil {
				t.Fatal("maintenance not released", e)
			}
			release()
		})
	}
}
