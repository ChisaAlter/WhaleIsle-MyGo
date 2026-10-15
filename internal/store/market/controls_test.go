package market

import (
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func TestBundleControlsHonorHostState(t *testing.T) {
	on, off := true, false
	tests := []struct {
		name string
		row Entry
		want bool
	}{
		{"live", Entry{IdentityKnown: true, Enabled: &on}, true},
		{"disabled", Entry{IdentityKnown: true, Enabled: &off}, true},
		{"unknown", Entry{Enabled: &on}, false},
		{"read-only", Entry{IdentityKnown: true, Enabled: &on, ReadOnlyReason: "protected"}, false},
		{"broken-disabled", Entry{IdentityKnown: true, Enabled: &off, Error: "missing"}, false},
		{"broken-enabled-can-disable", Entry{IdentityKnown: true, Enabled: &on, Error: "failed"}, true},
	}
	for _, test := range tests {
		if got := test.row.CanToggle(); got != test.want {
			t.Errorf("%s: CanToggle() = %v, want %v", test.name, got, test.want)
		}
	}
	rows := entriesFromBundles(api.ListBundlesResult{{Name: "locked", Enabled: true, ReadOnlyReason: func() *string { v := "protected"; return &v }(), Error: &api.ChangeResultError{Code: "broken"}}})
	if rows[0].ReadOnlyReason != "protected" || rows[0].Error != "broken" {
		t.Fatalf("Host restrictions lost: %+v", rows)
	}
	s := &Store{client: &dsh.Client{}, rows: []Entry{{Label: "live", IdentityKnown: true, Enabled: &on}}}
	if err := s.ValidateBundleChange("live", false); err != nil {
		t.Fatal(err)
	}
	if err := s.ValidateBundleChange("live", true); err == nil {
		t.Fatal("non-removable bundle accepted")
	}
	s.loading = true
	if err := s.ValidateBundleChange("live", false); err == nil {
		t.Fatal("stale refreshing inventory accepted")
	}
}
