package errcopy

import (
	"errors"
	"net"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/ui/i18n"
)

func TestSentenceAndUnderUseTypedErrorsAndHTTPStatus(t *testing.T) {
	withLocale(t, i18n.LocaleEN)

	cases := []struct {
		name   string
		err    error
		wantIn string
	}{
		{
			name:   "endpoint not ready sentinel",
			err:    dsh.ErrEndpointNotReady,
			wantIn: "timed out",
		},
		{
			name:   "rpc missing credential code",
			err:    &dsh.Error{Code: "MISSING_CREDENTIAL", Message: "opaque backend msg"},
			wantIn: "API key",
		},
		{
			name:   "http 401",
			err:    &dsh.HTTPError{Op: "session/list", Status: 401, Body: "unauthorized raw"},
			wantIn: "401",
		},
		{
			name:   "http 429",
			err:    &dsh.HTTPError{Op: "session/prompt", Status: 429, Body: "slow down"},
			wantIn: "429",
		},
		{
			name:   "http 502",
			err:    &dsh.HTTPError{Op: "session/list", Status: 502, Body: "bad gateway"},
			wantIn: "502",
		},
		{
			name:   "local dsh http 403 directory access denied maps to permission",
			err:    &dsh.HTTPError{Op: "directoryPicker/list", Status: 403, Body: "directory access denied"},
			wantIn: "permission",
		},
		{
			name:   "transport error",
			err:    &dsh.TransportError{Op: "session/list", Err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("opaque")}},
			wantIn: "network",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := Sentence(tc.err)
			if !strings.Contains(strings.ToLower(s), strings.ToLower(tc.wantIn)) {
				t.Fatalf("Sentence(%T) = %q, want substring %q", tc.err, s, tc.wantIn)
			}
			u := Under("Loading sessions", tc.err)
			if !strings.Contains(u, "Loading sessions") || !strings.Contains(strings.ToLower(u), strings.ToLower(tc.wantIn)) {
				t.Fatalf("Under(%T) = %q, want lead + %q", tc.err, u, tc.wantIn)
			}
			if strings.Contains(u, "did not finish") {
				t.Fatalf("Under(%T) fell through to CauseUnknown: %q", tc.err, u)
			}
		})
	}
}
