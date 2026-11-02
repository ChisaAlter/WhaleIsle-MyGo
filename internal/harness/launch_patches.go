package harness

import (
	"context"
	"fmt"
)

// readLaunchPatches snapshots desktop-owned optional overlays for this attempt.
// It copies both slices; changing future config never mutates an owned launch.
func readLaunchPatches(ctx context.Context, launch LaunchOptions, read func(context.Context) ([]string, error)) (LaunchOptions, error) {
	launch.PatchFiles = append([]string(nil), launch.PatchFiles...)
	if read == nil || launch.SkipUserPlugins {
		return launch, nil
	}
	if err := ctx.Err(); err != nil {
		return launch, err
	}
	patches, err := read(ctx)
	if err != nil {
		return launch, fmt.Errorf("desktop launch patches: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return launch, err
	}
	launch.PatchFiles = append(launch.PatchFiles, patches...)
	return launch, nil
}
