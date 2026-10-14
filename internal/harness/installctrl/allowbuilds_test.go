package installctrl

import "testing"

func TestParseAllowBuilds_ignoredScripts(t *testing.T) {
	log := `ERR_PNPM ignored build scripts: esbuild, sharp@0.32.0`
	got := ParseAllowBuilds(log)
	if len(got) != 2 {
		t.Fatalf("ParseAllowBuilds = %v", got)
	}
	seen := map[string]bool{got[0]: true, got[1]: true}
	if !seen["esbuild"] || !seen["sharp"] {
		t.Fatalf("ParseAllowBuilds = %v", got)
	}
}

func TestParseAllowBuilds_gitHosted(t *testing.T) {
	log := `git-hosted package "foo@1.0.0" needs to execute build scripts`
	got := ParseAllowBuilds(log)
	if len(got) != 1 || got[0] != "foo" {
		t.Fatalf("ParseAllowBuilds = %v", got)
	}
}

func TestIsBuildApprovalFailure(t *testing.T) {
	if !IsBuildApprovalFailure(1, []string{"esbuild"}) {
		t.Fatal("expected build approval failure")
	}
	if IsBuildApprovalFailure(0, []string{"esbuild"}) {
		t.Fatal("exit 0 should not need approval")
	}
	if IsBuildApprovalFailure(1, nil) {
		t.Fatal("empty allowBuilds should not need approval")
	}
}
