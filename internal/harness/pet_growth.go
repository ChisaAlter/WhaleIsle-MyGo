package harness

import (
	"context"
	"whaleisle/internal/petgrowth"
)

// ScanPetUsage reuses the configured runtime resolver without launching a Host,
// model or package installer. home is the native application's DSH data home.
func ScanPetUsage(ctx context.Context, node, home string) (map[string]int64, error) {
	bin, err := resolveNode(node)
	if err != nil {
		return nil, err
	}
	return petgrowth.Scan(ctx, bin, home)
}

// ScanPetUsageSources is the source-aware desktop entry point. Legacy callers
// retain ScanPetUsage's sessions-only contract.
func ScanPetUsageSources(ctx context.Context, node, home, usageFile string) (petgrowth.Usage, error) {
	bin, err := resolveNode(node)
	if err != nil {
		return petgrowth.Usage{}, err
	}
	return petgrowth.ScanUsage(ctx, bin, home, usageFile)
}
