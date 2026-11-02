package projectpreference

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestProjectPreferenceWaitsForProtectedRuntimeAndKeepsOtherState(t *testing.T) {
	for _, next := range []bool{true, false} {
		c := new(Controller)
		value := !next
		var order []string
		t.Run(map[bool]string{true: "enable", false: "disable"}[next], func(t *testing.T) {
			err := c.Set(context.Background(), next, Transition{
				Read: func() bool { return value }, Write: func(expected, next bool) error {
					if value != expected {
						t.Fatal("write lost compare")
					}
					order = append(order, "write")
					value = next
					return nil
				},
				Acquire: func() (func(), error) {
					order = append(order, "lease")
					return func() { order = append(order, "release") }, nil
				},
				LeaveLead: func(context.Context) error { order = append(order, "leave-ready"); return nil },
				Protect: func(_ context.Context, commit func() error) (bool, error) {
					order = append(order, "protect")
					if !c.Snapshot(value).Busy {
						t.Fatal("switch active during confirmation")
					}
					err := commit()
					return err == nil, err
				},
				Restart: func(context.Context) error {
					order = append(order, "restart-ready")
					if v := c.Snapshot(value); !v.Busy || v.Enabled != next {
						t.Fatal("pending value lost")
					}
					return nil
				},
			})
			want := []string{"lease", "protect", "write", "restart-ready", "release"}
			if !next {
				want = []string{"lease", "leave-ready", "protect", "write", "restart-ready", "release"}
			}
			if err != nil || value != next || !reflect.DeepEqual(order, want) || c.Snapshot(value).Busy {
				t.Fatalf("value%v order%v err%v", value, order, err)
			}
		})
	}
}

func TestProjectPreferenceCancelLeaveWriteAndRestartFailures(t *testing.T) {
	for _, failure := range []string{"lease", "leave", "cancel", "write", "restart"} {
		t.Run(failure, func(t *testing.T) {
			c := new(Controller)
			value := true
			writes, restarts := 0, 0
			err := c.Set(context.Background(), false, Transition{
				Read: func() bool { return value },
				Write: func(expected, next bool) error {
					writes++
					if value != expected {
						t.Fatal("rollback overwrites changed field")
					}
					if failure == "write" {
						return errors.New("disk")
					}
					value = next
					return nil
				},
				Acquire: func() (func(), error) {
					if failure == "lease" {
						return nil, errors.New("maintenance")
					}
					return func() {}, nil
				},
				LeaveLead: func(context.Context) error {
					if failure == "leave" {
						return errors.New("follow")
					}
					return nil
				},
				Protect: func(_ context.Context, commit func() error) (bool, error) {
					if failure == "cancel" {
						return false, nil
					}
					err := commit()
					return err == nil, err
				},
				Restart: func(context.Context) error {
					restarts++
					if failure == "restart" {
						return errors.New("runtime")
					}
					return nil
				},
			})
			v := c.Snapshot(value)
			if err == nil || !value || v.Busy || v.Error == "" {
				t.Fatalf("false accepted on failure: %+v %v", v, err)
			}
			if failure == "cancel" && (!v.RestartCancelled || writes != 0 || restarts != 0) {
				t.Fatal("cancel had side effects")
			}
			if failure == "restart" && (writes != 2 || restarts != 1) {
				t.Fatal("restart failure did not roll back once")
			}
			if (failure == "lease" || failure == "leave") && (writes != 0 || restarts != 0) {
				t.Fatal("prewrite failure had side effects")
			}
		})
	}
}

func TestProjectPreferenceRejectsDoubleInputWhileWaiting(t *testing.T) {
	c := new(Controller)
	value := false
	blocked := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	transition := Transition{Read: func() bool { return value }, Write: func(_, n bool) error { value = n; return nil }, Protect: func(_ context.Context, commit func() error) (bool, error) {
		close(blocked)
		<-release
		err := commit()
		return err == nil, err
	}, Restart: func(context.Context) error { return nil }}
	go func() { done <- c.Set(context.Background(), true, transition) }()
	<-blocked
	if err := c.Set(context.Background(), false, transition); !errors.Is(err, ErrBusy) {
		t.Fatalf("duplicate input accepted:%v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !value || c.Snapshot(value).Busy {
		t.Fatal("first transaction lost")
	}
}
