//go:build windows

package update

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPowerShellEnvIsolation(t *testing.T) {
	tmp := t.TempDir()
	env := isolatedPowerShellEnv([]string{
		"PATH=C:\\Windows\\System32",
		"PSModuleAnalysisCachePath=C:\\repo\\bad\\ModuleAnalysisCache",
	}, tmp)

	var cacheVal string
	var hasTemp bool
	for _, entry := range env {
		if k, v, ok := strings.Cut(entry, "="); ok {
			if strings.EqualFold(k, "PSModuleAnalysisCachePath") {
				cacheVal = v
			}
			if strings.EqualFold(k, "TEMP") {
				hasTemp = true
			}
		}
	}
	if !strings.HasPrefix(strings.ToLower(cacheVal), strings.ToLower(filepath.Clean(tmp))) {
		t.Fatalf("expected isolated PSModuleAnalysisCachePath inside %q, got %q", tmp, cacheVal)
	}
	if !hasTemp {
		t.Fatal("expected fallback TEMP inside isolated env")
	}
}
