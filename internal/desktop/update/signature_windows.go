//go:build windows

package update

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"whaleisle/internal/processutil"
)

func isolatedPowerShellEnv(base []string, tempDir string) []string {
	cacheFile := filepath.Join(tempDir, "ModuleAnalysisCache")
	env := make([]string, 0, len(base)+4)
	hasTemp, hasTmp, hasLocalAppData := false, false, false
	for _, entry := range base {
		k, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		switch {
		case strings.EqualFold(k, "PSModuleAnalysisCachePath"):
			continue
		case strings.EqualFold(k, "TEMP"):
			hasTemp = true
		case strings.EqualFold(k, "TMP"):
			hasTmp = true
		case strings.EqualFold(k, "LOCALAPPDATA"):
			hasLocalAppData = true
		}
		env = append(env, entry)
	}
	env = append(env, "PSModuleAnalysisCachePath="+cacheFile)
	if !hasTemp {
		env = append(env, "TEMP="+tempDir)
	}
	if !hasTmp {
		env = append(env, "TMP="+tempDir)
	}
	if !hasLocalAppData {
		env = append(env, "LOCALAPPDATA="+tempDir)
	}
	return env
}

// probeAuthenticode uses PowerShell Get-AuthenticodeSignature (no extra deps).
func probeAuthenticode(ctx context.Context, path string) (SignatureInfo, error) {
	psPath := strings.ReplaceAll(path, "'", "''")
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
# Go-launched Windows PowerShell can inherit PowerShell 7 module search paths.
# Load its own built-in module, not an incompatible module found earlier there.
Import-Module (Join-Path $PSHOME 'Modules\Microsoft.PowerShell.Security\Microsoft.PowerShell.Security.psd1') -ErrorAction Stop
$s = Get-AuthenticodeSignature -LiteralPath '%s'
$pub = ''
if ($null -ne $s.SignerCertificate) { $pub = [string]$s.SignerCertificate.Subject }
Write-Output ('STATUS=' + $s.Status.ToString())
Write-Output ('PUBLISHER=' + $pub)
`, psPath)

	psTempDir, tmpErr := os.MkdirTemp("", "whaleisle-ps-")
	if tmpErr == nil {
		defer os.RemoveAll(psTempDir)
	} else {
		psTempDir = os.TempDir()
	}
	cmd := exec.CommandContext(ctx, "powershell.exe",
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-Command", script)
	cmd.Env = isolatedPowerShellEnv(os.Environ(), psTempDir)
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
