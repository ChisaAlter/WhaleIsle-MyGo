// Package externallauncher manages a separately registered native desktop.
package externallauncher

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"whaleisle/internal/desktop/update"
)

const StopArgument = "--managed-launcher-stop"

var ErrNotInstalled = errors.New("no verified native desktop installation")

type Installation struct{ Root, Executable, Version string }
type Dependencies struct {
	Discover    func() (Installation, error)
	Running     func(Installation) (bool, error)
	Start       func(Installation) error
	RequestStop func(Installation) error
	Maintenance func() (func(), error)
	Install     func(context.Context, update.ApplyResult) error
}
type Controller struct {
	mu   sync.Mutex
	deps Dependencies
}

func New(deps Dependencies) *Controller { return &Controller{deps: deps} }
func (c *Controller) Snapshot() (Installation, bool, error) {
	i, err := c.deps.Discover()
	if err != nil {
		return i, false, err
	}
	running, err := c.deps.Running(i)
	return i, running, err
}
func (c *Controller) admit() (func(), error) {
	if !c.mu.TryLock() {
		return nil, errors.New("another launcher operation is active")
	}
	release, err := c.deps.Maintenance()
	if err != nil {
		c.mu.Unlock()
		return nil, err
	}
	return func() { release(); c.mu.Unlock() }, nil
}
func (c *Controller) Start(ctx context.Context) error {
	release, err := c.admit()
	if err != nil {
		return err
	}
	defer release()
	i, running, err := c.Snapshot()
	if err != nil {
		return err
	}
	if running {
		return nil
	}
	if err = c.deps.Start(i); err != nil {
		return err
	}
	return c.wait(ctx, i, true)
}
func (c *Controller) Stop(ctx context.Context) error {
	release, err := c.admit()
	if err != nil {
		return err
	}
	defer release()
	i, running, err := c.Snapshot()
	if err != nil {
		return err
	}
	if !running {
		return nil
	}
	if err = c.deps.RequestStop(i); err != nil {
		return err
	}
	return c.wait(ctx, i, false)
}
func (c *Controller) wait(ctx context.Context, i Installation, want bool) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		running, err := c.deps.Running(i)
		if err != nil {
			return err
		}
		if running == want {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("desktop did not reach requested state; it may have declined task protection or require a newer version")
		case <-ticker.C:
		}
	}
}

// Install requires the desktop already stopped through its protected quit path.
// Installer completion alone is insufficient: registration must match the chosen version.
func (c *Controller) Install(ctx context.Context, staged update.ApplyResult, version string) (Installation, error) {
	release, err := c.admit()
	if err != nil {
		return Installation{}, err
	}
	defer release()
	_, running, err := c.Snapshot()
	if err != nil && !errors.Is(err, ErrNotInstalled) {
		return Installation{}, err
	}
	if running {
		return Installation{}, errors.New("stop the desktop through its protected quit flow before installing")
	}
	if strings.TrimSpace(version) == "" {
		return Installation{}, errors.New("missing target version")
	}
	if err = c.deps.Install(ctx, staged); err != nil {
		return Installation{}, err
	}
	fresh, err := c.deps.Discover()
	if err != nil {
		return Installation{}, err
	}
	norm := func(s string) string { return strings.TrimPrefix(strings.TrimPrefix(s, "mygo-"), "v") }
	if norm(fresh.Version) != norm(version) {
		return Installation{}, errors.New("installer finished without registering the selected desktop version")
	}
	return fresh, nil
}
