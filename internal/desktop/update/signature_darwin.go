//go:build darwin

package update

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// The shared signature options also apply to native macOS packages. A probe is
// invoked only when the existing RequireSignature/ExpectedPublisher policy asks
// for it. This reads the signature; it never signs or changes Gatekeeper policy.
func probeAuthenticode(ctx context.Context, path string) (SignatureInfo, error) {
	if err := ctx.Err(); err != nil {
		return SignatureInfo{}, err
	}
	output, err := exec.CommandContext(ctx, "/usr/bin/codesign", "--verify", "--strict", path).CombinedOutput()
	if ctx.Err() != nil {
		return SignatureInfo{}, ctx.Err()
	}
	if err != nil {
		return SignatureInfo{Status: SignatureInvalid, Detail: strings.TrimSpace(string(output))}, fmt.Errorf("macOS code signature: %w", err)
	}
	output, err = exec.CommandContext(ctx, "/usr/bin/codesign", "--display", "--verbose=4", path).CombinedOutput()
	if ctx.Err() != nil {
		return SignatureInfo{}, ctx.Err()
	}
	if err != nil {
		return SignatureInfo{}, fmt.Errorf("macOS signature identity: %w", err)
	}
	var publishers []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Authority=") {
			publishers = append(publishers, strings.TrimPrefix(line, "Authority="))
		}
		if strings.HasPrefix(line, "TeamIdentifier=") && line != "TeamIdentifier=not set" {
			publishers = append(publishers, strings.TrimPrefix(line, "TeamIdentifier="))
		}
	}
	if len(publishers) == 0 {
		return SignatureInfo{Status: SignatureNone, Detail: "macOS package has no signing identity"}, nil
	}
	return SignatureInfo{Status: SignatureValid, Publisher: strings.Join(publishers, "; "), Detail: "macOS code signature verified"}, nil
}
