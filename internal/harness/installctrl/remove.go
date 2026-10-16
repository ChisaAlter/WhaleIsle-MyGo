package installctrl

import (
	"context"
	"fmt"
	"strings"
)

// RemovePluginContext uses only the bundled CLI, never a network-fetched dsh.
// The caller owns import exclusion, task protection and stopping the Host.
func RemovePluginContext(parent context.Context, cfg InstallConfig, name string) Result {
	out := Result{Spec: name}
	if RegistryPluginName(name+"@0.0.0") != name {
		out.Error = "invalid plugin name"
		return out
	}
	if !InstallReady(cfg) || strings.TrimSpace(cfg.ProfileDir) == "" {
		out.Error = "bundled Node and dsh CLI are required for removal"
		return out
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultInstallTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	runner := cfg.Runner
	if runner == nil {
		runner = execPluginRunner(cfg)
	}
	code, log, err := runner(ctx, cfg.ProfileDir, cfg.NodeBin, harnessBinJS(cfg.HarnessRoot), "plugin", "--profile", WebProfile, "remove", name)
	out.Log = log
	if err != nil || code != 0 {
		out.Error = fmt.Sprintf("plugin removal failed (exit %d): %v", code, err)
		return out
	}
	out.OK = true
	return out
}
