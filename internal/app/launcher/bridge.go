package launcherctl

import (
	"context"

	"whaleisle/internal/app/updater"
	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
)

// WindowRef identifies a host shell window without importing toolkit types.
type WindowRef any

// HostEnv captures a consistent point-in-time snapshot of shell state needed by
// launcher operations so launcherctl locks never nest with mainState locks.
type HostEnv struct {
	BuildVersion       string
	DownloadRoute      string
	Quitting           bool
	LauncherImportBusy bool
	TaskRestartPending bool
	HostStarting       bool
	HasSupervisor      bool
	HasSessions        bool
	BootStatus         harness.Status
	BootLogs           []string
	StartLogs          []string
	ConfigStore        *config.Store
	MainWindow         WindowRef
	ExclusivePanelOpen bool
	HasOccupiedDrafts  bool
	AssistantPanelGen  uint64
	WorkspaceOpenGen   uint64
}

// NativePathSetter applies isolated app identity and data paths on host boot.
type NativePathSetter interface {
	SetName(name string)
	SetUserDataPath(path string)
	SetCachePath(path string)
	SetLogsPath(path string)
}

// HostDeps abstracts shell, task-protection, and sibling-controller hooks.
// Launcher dependencies flow strictly outward or down to updater (*updater.Center);
// updater never imports launcherctl.
type HostDeps interface {
	Env() HostEnv
	LifecycleContext() context.Context
	Updater() *updater.Center

	MarketBusy() bool
	TryAcquireRepairSlot() (release func(), ok bool)
	TrySetTaskRestartPending(pending bool) bool

	ConfigureGenericComponents()
	SyncHarnessFromConfig()
	StartHarnessWithConfig(cfg config.Config)
	RestartSupervisorIfPresent()
	StopSupervisorIfPresent()
	HasCurrentSupervisor() bool

	WithTaskProtection(win WindowRef, reason string, fn func() error) bool
	CloseTaskProtectedResources() error

	OpenStartupBlankSession(win WindowRef, sessions *session.Store, hostGen uint64, workspaceID string)
	ObserveBridgeAlerts()
	RefuseHostAction()

	UpdateWindow(win WindowRef, fn func())
	InvalidateSurface(win WindowRef)
}
