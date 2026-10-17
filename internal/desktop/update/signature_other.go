//go:build !windows

package update

import (
	"context"
	"runtime"
)

// probeAuthenticode is a documented no-op outside Windows: Authenticode is
// unavailable, so verification is skipped (success) rather than failing CI/host
// builds on other GOOS values.
func probeAuthenticode(ctx context.Context, path string) (SignatureInfo, error) {
	if err := ctx.Err(); err != nil {
		return SignatureInfo{}, err
	}
	_ = path
	return SignatureInfo{
		Status: SignatureSkipped,
		Detail: "Authenticode unavailable on " + runtime.GOOS,
	}, nil
}
