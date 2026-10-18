package update

import (
	"strings"
	"testing"
)

func TestNativeReleasePromptReviewOnly(t *testing.T) {
	full := NativeRelease{Feed: Feed{Version: "2.0", Notes: "fixture release notes", Size: 2 << 20}, Tag: "v2", Platform: "windows", Arch: "amd64", PackageKind: "full"}
	delta := full
	delta.PackageKind = "delta"
	delta.Delta = &NativeDelta{BaseVersion: "1.0", DownloadSize: 1 << 20}
	blocked := delta
	blocked.Delta = &NativeDelta{BaseVersion: "blocked"}
	blocked.UnavailableReason = "missing base"
	got := NativeReleasePrompt("1.0", full, []NativeRelease{full, delta, blocked}, false)
	for _, want := range []string{"Current version: 1.0", "Available version: 2.0", "Full", "Delta (from 1.0)", "2.0 MiB", "fixture release notes", "Nothing is downloaded or installed now"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(got, "blocked") {
		t.Fatal("unavailable delta advertised")
	}
	full.Notes = strings.Repeat("鲸", 3100)
	got = NativeReleasePrompt("1.0", full, nil, true)
	if !strings.Contains(got, "完整说明见版本页") {
		t.Fatal("long notes truncation not disclosed")
	}
}
