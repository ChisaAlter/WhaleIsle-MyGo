package market

import (
	"context"
	"path/filepath"
	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/paths"
)

func RemovePluginContext(ctx context.Context, name string, opts InstallOptions) installctrl.Result {
	node, err := resolveNode(opts.NodeBin)
	if err != nil {
		return installctrl.Result{Error: err.Error()}
	}
	home := opts.DshHome
	if home == "" {
		home = paths.DshHome()
	}
	return installctrl.RemovePluginContext(ctx, installctrl.InstallConfig{NodeBin: node, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), HarnessRoot: paths.HarnessRoot()}, name)
}
