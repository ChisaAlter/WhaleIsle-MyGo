//go:build windows

package update

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"whaleisle/internal/processutil"
)

// probeAuthenticode uses PowerShell Get-AuthenticodeSignature (no extra deps).
func probeAuthenticode(ctx context.Context, path string) (SignatureInfo, error) {
	psPath := strings.ReplaceAll(path, "'", "''")
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$s = Get-AuthenticodeSignature -LiteralPath '%s'
$pub = ''
if ($null -ne $s.SignerCertificate) { $pub = [string]$s.SignerCertificate.Subject }
Write-Output ('STATUS=' + $s.Status.ToString())
Write-Output ('PUBLISHER=' + $pub)
`, psPath)

	cmd := exec.CommandContext(ctx, "powershell.exe",
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-Command", script)
	processutil.Background(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return SignatureInfo{}, fmt.Errorf("Get-AuthenticodeSignature: %s", msg)
	}
	return parseAuthenticodeOutput(stdout.String()), nil
}
