package harness

import (
	"fmt"

	"whaleisle/internal/paths"
	"whaleisle/internal/startuptrace"
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
	// Reject a stale or mixed reader before preparing anything in the shared
	// desktop profile. Resolving is read-only; patches are added after Prepare.
	endValidate := startuptrace.Begin("runtime.validate")
	if _, err := Resolve(o, port); err != nil {
		endValidate()
		return StartPlan{}, err
	}
	endValidate()
	if err := paths.EnsureDshHome(); err != nil {
		return StartPlan{}, fmt.Errorf("dsh-home: %w", err)
	}
	endPrepare := startuptrace.Begin("runtime.profile_prepare")
	patches, err := Prepare(prep)
	endPrepare()
	if err != nil {
		return StartPlan{}, err
	}
	o.PatchFiles = append(append([]string(nil), o.PatchFiles...), patches...)
	endResolve := startuptrace.Begin("runtime.launch_resolve")
	plan, err := Resolve(o, port)
	endResolve()
	if err != nil {
		return StartPlan{}, err
	}
	return StartPlan{Plan: plan, PatchFiles: o.PatchFiles}, nil
}
