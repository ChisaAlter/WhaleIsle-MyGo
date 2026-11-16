package harness

import (
	"fmt"
	"os"
	"path/filepath"

	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/paths"
	"whaleisle/internal/startuptrace"
)

// StartPlan combines desktop plugin preparation with launcher resolution.
type StartPlan struct {
	Plan
	PatchFiles []string
}

// PlanStart prepares overlays/junctions under the web profile and resolves
// a vendored (or fallback) launch plan for the given port in a single Resolve pass.
func PlanStart(o LaunchOptions, prep PrepareOptions, port int) (StartPlan, error) {
	if o.Home == "" {
		o.Home = paths.DshHome()
	}
	if o.VendorDir == "" {
		o.VendorDir = paths.VendorDir()
	}
	if o.HarnessRoot == "" {
		o.HarnessRoot = paths.HarnessRoot()
	}
	if prep.VendorDir == "" {
		prep.VendorDir = o.VendorDir
	}
	if prep.ProfileDir == "" {
		prep.ProfileDir = paths.WebProfileDir()
	}
	if prep.DshHome == "" {
		prep.DshHome = o.Home
	}
	if prep.HarnessRoot == "" {
		prep.HarnessRoot = o.HarnessRoot
	}
	prep.SkipUserPlugins = prep.SkipUserPlugins || o.SkipUserPlugins
	// Reject a stale or mixed reader before preparing anything in the shared
	// desktop profile. Resolving once validates the launcher/runtime; patches and
	// desktop environment entries are then applied to the resolved plan without
	// re-running runtime hashing or registry verification.
	endValidate := startuptrace.Begin("runtime.validate")
	plan, resolvedOpts, err := resolveLaunchPlanWithOptions(o, port, os.ReadFile)
	endValidate()
	if err != nil {
		return StartPlan{}, err
	}
	if err := paths.EnsureDshHome(); err != nil {
		return StartPlan{}, fmt.Errorf("dsh-home: %w", err)
	}
	endPrepare := startuptrace.Begin("runtime.profile_prepare")
	patches, err := Prepare(prep)
	endPrepare()
	if err != nil {
		return StartPlan{}, err
	}
	o = resolvedOpts
	for _, p := range patches {
		if p == "" {
			continue
		}
		abs, absErr := filepath.Abs(p)
		if absErr != nil {
			return StartPlan{}, fmt.Errorf("resolve patch path: %w", absErr)
		}
		o.PatchFiles = append(o.PatchFiles, abs)
	}
	o.DesktopDefaults = true
	o.WhaleUIHost, err = prepareWhaleUIHost(o.Home)
	if err != nil {
		return StartPlan{}, fmt.Errorf("Whale UI host: %w", err)
	}
	plan.Args = buildLaunchArgs(o, port, plan.prefixArgs, plan.preloadRoot)
	plan.Env = childEnv(o)
	return StartPlan{Plan: plan, PatchFiles: o.PatchFiles}, nil
}

func prepareWhaleUIHost(home string) (string, error) {
	path := filepath.Join(home, "runtime", "whale-ui", "host.cjs")
	data := openui.WhaleHostValidator()
	if old, err := os.ReadFile(path); err == nil && string(old) == string(data) {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := writeFileAtomic(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
