package harness

import (
	"fmt"

	"whaleisle/internal/paths"
)

// StartPlan combines desktop plugin preparation with launcher resolution.
type StartPlan struct {
	Plan
	PatchFiles []string
}

// PlanStart prepares overlays/junctions under the web profile and resolves
// a vendored (or fallback) launch plan for the given port.
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
	if err := paths.EnsureDshHome(); err != nil {
		return StartPlan{}, fmt.Errorf("dsh-home: %w", err)
	}
	patches, err := Prepare(prep)
	if err != nil {
		return StartPlan{}, err
	}
	o.PatchFiles = append(append([]string(nil), o.PatchFiles...), patches...)
	plan, err := Resolve(o, port)
	if err != nil {
		return StartPlan{}, err
	}
	return StartPlan{Plan: plan, PatchFiles: o.PatchFiles}, nil
}
