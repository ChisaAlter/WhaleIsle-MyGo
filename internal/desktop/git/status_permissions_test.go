package git

import "testing"

func TestRepositoryStatusPushAndPRPermissions(t *testing.T) {
	base := RepositoryStatus{IsRepo: true, Branch: "feature", HasOrigin: true, Ahead: 1, AheadOfDefault: 1}
	if !base.CanPush() || !base.CanCreatePR() {
		t.Fatal("publishable feature branch disabled")
	}
	for _, tc := range []struct {
		name     string
		change   func(*RepositoryStatus)
		push, pr bool
	}{
		{"dirty", func(s *RepositoryStatus) { s.Dirty = true }, true, false},
		{"behind", func(s *RepositoryStatus) { s.Behind = 1 }, false, false},
		{"detached", func(s *RepositoryStatus) { s.Branch = "" }, false, false},
		{"no remote", func(s *RepositoryStatus) { s.HasOrigin = false }, false, false},
		{"no commits ahead", func(s *RepositoryStatus) { s.Ahead = 0 }, false, true},
		{"no default changes", func(s *RepositoryStatus) { s.AheadOfDefault = 0 }, true, false},
		{"count unavailable", func(s *RepositoryStatus) { s.Ahead = 0; s.AheadUnreliable = true }, true, true},
		{"not repository", func(s *RepositoryStatus) { s.IsRepo = false }, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := base
			tc.change(&s)
			if s.CanPush() != tc.push || s.CanCreatePR() != tc.pr {
				t.Fatalf("status=%+v push=%v pr=%v", s, s.CanPush(), s.CanCreatePR())
			}
		})
	}
}

func TestBranchInventoryKeepsUnsupportedNamesVisibleButDisabled(t *testing.T) {
	branches := parseBranches("refs/heads/main\t*\t\t.\nrefs/heads/feature/topic\t \t\t.\nrefs/heads/分支\t \t\t.\nrefs/remotes/origin/main\t \t\t.\nrefs/remotes/origin/HEAD\t \trefs/remotes/origin/main\t.\n")
	if len(branches) != 3 {
		t.Fatalf("inventory=%+v", branches)
	}
	seen := map[string]Branch{}
	for _, b := range branches {
		seen[b.Name] = b
	}
	if !seen["main"].Current || seen["feature/topic"].Unsupported || !seen["分支"].Unsupported {
		t.Fatalf("branch capabilities=%+v", branches)
	}
	for _, name := range []string{"-option", "feature..topic", "topic.lock", "topic/"} {
		if supportedBranchName(name) {
			t.Fatalf("unsafe branch selectable: %q", name)
		}
	}
}
