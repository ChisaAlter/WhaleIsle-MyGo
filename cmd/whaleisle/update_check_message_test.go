package main

import (
	"errors"
	"strings"
	"testing"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/i18n"
)

func TestUpdateCheckMessageUsesProductCopy(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	cases := []struct {
		name string
		res  desktopupdate.Result
		err  error
		want string
		deny string
	}{
		{"unconfigured", desktopupdate.Result{Status: desktopupdate.StatusNotConfigured, Message: "MyGo updater feed not configured"}, nil, "no update feed configured", "not_configured"},
		{"latest", desktopupdate.Result{Status: desktopupdate.StatusUpToDate, Version: "1.2.3"}, nil, "latest version", "up_to_date"},
		{"available", desktopupdate.Result{Status: desktopupdate.StatusUpdateAvailable, Version: "2.0.0", ReleaseURL: "https://example.com/r"}, nil, "2.0.0", "update_available"},
		{"failure", desktopupdate.Result{}, errors.New("network down"), "network down", ""},
	}
	for _, tc := range cases {
		got := updateCheckMessage(tc.res, tc.err)
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: message %q lacks %q", tc.name, got, tc.want)
		}
		if tc.deny != "" && strings.Contains(got, tc.deny) {
			t.Errorf("%s: raw status %q leaked: %q", tc.name, tc.deny, got)
		}
	}
}
