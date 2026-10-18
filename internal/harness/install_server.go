package harness

import (
	"context"
	"net/http"
	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/paths"
)

// InstallServerOptions configures the loopback install control plane.
type InstallServerOptions struct {
	Project     func(context.Context, map[string]any) map[string]any
	HTMLPreview http.Handler
	NodeBin     string
	OnRestart   func()
	// Desktop wires the optional authenticated /desktop/* control callbacks.
	// Leaving it zero-valued preserves the legacy routes.
	Desktop installctrl.DesktopOps
}

// NewInstallServer returns a loopback install control server wired to restart
// the harness after a successful install. When Node.js and the vendored dsh CLI
// are available, installs run via dsh plugin add (pnpm in the web profile);
// otherwise requests receive a clear error.
func NewInstallServer(opts InstallServerOptions) *installctrl.Server {
	cfg := installctrl.InstallConfig{
		ProfileDir:  paths.WebProfileDir(),
		DshHome:     paths.DshHome(),
		HarnessRoot: paths.HarnessRoot(),
	}
	if node, err := resolveNode(opts.NodeBin); err == nil {
		cfg.NodeBin = node
	}

	var install installctrl.InstallFunc
	if installctrl.InstallReady(cfg) {
		install = installctrl.PNPMInstall(cfg)
	} else {
		install = func(spec string, _ installctrl.InstallOpts) installctrl.Result {
			msg := "desktop plugin install requires Node.js and a built vendored harness (setup-harness)"
			if _, err := resolveNode(opts.NodeBin); err != nil {
				msg = err.Error()
			}
			return installctrl.Result{
				OK:    false,
				Spec:  spec,
				Error: msg,
			}
		}
	}

	return installctrl.New(installctrl.Options{
		OnRestart:   opts.OnRestart,
		Project:     opts.Project,
		HTMLPreview: opts.HTMLPreview,
		Install:     install,
		Desktop:     opts.Desktop,
	})
}
