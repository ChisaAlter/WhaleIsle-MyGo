package marketctl

import (
	"context"

	"whaleisle/internal/config"
)

// HostEnvSnapshot carries read-only shell/harness readiness flags needed by
// market coordination without holding marketctl locks across shell mutexes.
type HostEnvSnapshot struct {
	LauncherImportBusy bool
	Quitting           bool
	TaskRestartPending bool
	LauncherStartBusy  bool
	HostStarting       bool
	HasSupervisor      bool
	BootGeneration     int
	NodeBin            string
}

// HostDeps abstracts shell-level capabilities consumed by Coordinator.
type HostDeps interface {
	Env() HostEnvSnapshot
	LifecycleContext() context.Context
	GoPluginInstallRegistry() (*config.PluginInstallRegistry, error)
	SaveGoPluginInstallRegistry(choice *config.PluginInstallRegistry) error
	SyncHarnessAndRestart()
	UpdateWindow(fn func())
	InvalidateWindow()
	CloseWebViewHost()
}
