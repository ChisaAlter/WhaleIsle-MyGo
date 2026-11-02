package main

import "testing"

// TestSystemNotificationsAllowed pins the native policy: registered local
// builds and packages are supported, with isolated profiles gated separately.
func TestSystemNotificationsAllowed(t *testing.T) {
	cases := []struct {
		platform string
		dev      bool
		want     bool
	}{
		{"windows", true, true},
		{"windows", false, true},
		{"darwin", true, true},
		{"darwin", false, true},
		{"linux", true, true},
	}
	for _, c := range cases {
		if got := systemNotificationsAllowed(c.platform, c.dev); got != c.want {
			t.Fatalf("allowed(%q, dev=%v) = %v, want %v", c.platform, c.dev, got, c.want)
		}
	}
}

// TestDevelopmentAppMatchesThePackagingBoundary keeps the boundary tied to the
// same version value the packaging scripts set, so a release build is never
// mistaken for a development run.
func TestDevelopmentAppMatchesThePackagingBoundary(t *testing.T) {
	previous := buildVersion
	defer func() { buildVersion = previous }()
	for _, version := range []string{"", "dev", "0.0.0-dev"} {
		buildVersion = version
		if !developmentApp() {
			t.Fatalf("%q is not a development build", version)
		}
	}
	buildVersion = "1.4.2"
	if developmentApp() {
		t.Fatal("release build treated as development")
	}
}
