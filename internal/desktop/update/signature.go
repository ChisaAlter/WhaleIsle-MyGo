package update

import (
	"context"
	"fmt"
	"strings"
)

// SignatureStatus is the outcome of the platform signature probe.
type SignatureStatus string

const (
	// SignatureValid means the file has a valid platform signature.
	SignatureValid SignatureStatus = "valid"
	// SignatureSkipped means verification was not run (unsupported platform, or not requested).
	SignatureSkipped SignatureStatus = "skipped"
	// SignatureNone means the file is not signed.
	SignatureNone SignatureStatus = "not_signed"
	// SignatureInvalid means the signature is present but not Valid.
	SignatureInvalid SignatureStatus = "invalid"
)

// SignatureInfo is returned by VerifyAuthenticode / CheckSignature.
type SignatureInfo struct {
	Status    SignatureStatus
	Publisher string
	Detail    string
}

// VerifyOptions configures optional platform-signature enforcement after download.
type VerifyOptions struct {
	// RequireSignature, when true, fails closed unless the probe reports SignatureValid
	// on Windows or Darwin. Other platforms report SignatureSkipped; enforcement
	// retains the documented no-op for those unsupported targets.
	RequireSignature bool
	// ExpectedPublisher, when non-empty, requires the signer certificate subject to
	// contain this substring (case-insensitive). Implies a signature probe on Windows or Darwin.
	ExpectedPublisher string
}

// authenticodeProbe is the platform probe. Tests replace it with a stub.
var authenticodeProbe = probeAuthenticode

// wantsSignatureCheck reports whether opt requests a probe/enforcement step.
func wantsSignatureCheck(opt VerifyOptions) bool {
	return opt.RequireSignature || strings.TrimSpace(opt.ExpectedPublisher) != ""
}

// VerifyAuthenticode probes Authenticode on path (Windows via PowerShell
// Get-AuthenticodeSignature) or codesign on Darwin. Other platforms skip.
func VerifyAuthenticode(ctx context.Context, path string) (SignatureInfo, error) {
	if ctx.Err() != nil {
		return SignatureInfo{}, ctx.Err()
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return SignatureInfo{}, fmt.Errorf("empty path for authenticode check")
	}
	return authenticodeProbe(ctx, path)
}

// CheckSignature probes (when requested) and applies RequireSignature /
// ExpectedPublisher policy. When neither option is set it returns SignatureSkipped
// without probing.
func CheckSignature(ctx context.Context, path string, opt VerifyOptions) (SignatureInfo, error) {
	if ctx.Err() != nil {
		return SignatureInfo{}, ctx.Err()
	}
	if !wantsSignatureCheck(opt) {
		return SignatureInfo{Status: SignatureSkipped, Detail: "signature check not requested"}, nil
	}
	info, err := VerifyAuthenticode(ctx, path)
	if err != nil {
		// Fail closed when a check was requested but the probe could not run.
		return SignatureInfo{Status: SignatureInvalid, Detail: err.Error()}, fmt.Errorf("authenticode: %w", err)
	}
	if err := enforceSignature(info, opt); err != nil {
		return info, err
	}
	return info, nil
}

func enforceSignature(info SignatureInfo, opt VerifyOptions) error {
	if info.Status == SignatureSkipped {
		// Unsupported platform: documented no-op success.
		return nil
	}
	if opt.RequireSignature || strings.TrimSpace(opt.ExpectedPublisher) != "" {
		if info.Status != SignatureValid {
			detail := info.Detail
			if detail == "" {
				detail = string(info.Status)
			}
			return fmt.Errorf("authenticode verification failed: %s", detail)
		}
	}
	expect := strings.TrimSpace(opt.ExpectedPublisher)
	if expect != "" {
		pub := strings.ToLower(info.Publisher)
		if pub == "" || !strings.Contains(pub, strings.ToLower(expect)) {
			return fmt.Errorf("authenticode publisher %q does not match %q", info.Publisher, expect)
		}
	}
	return nil
}

// parseAuthenticodeOutput parses STATUS=/PUBLISHER= lines from the PowerShell probe.
func parseAuthenticodeOutput(out string) SignatureInfo {
	info := SignatureInfo{Status: SignatureInvalid, Detail: "unknown"}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "STATUS="):
			raw := strings.TrimPrefix(line, "STATUS=")
			info.Detail = raw
			switch strings.ToLower(raw) {
			case "valid":
				info.Status = SignatureValid
			case "notsigned":
				info.Status = SignatureNone
			default:
				info.Status = SignatureInvalid
			}
		case strings.HasPrefix(line, "PUBLISHER="):
			info.Publisher = strings.TrimPrefix(line, "PUBLISHER=")
		}
	}
	return info
}
