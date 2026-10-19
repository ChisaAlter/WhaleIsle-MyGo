package dsh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConnectionRecoveryBootstrapAuthority(t *testing.T) {
	assignment := func(value string) string {
		return `<script>globalThis["__DSH_CONNECTION_RECOVERY__"] = ` + value + `;</script>`
	}
	for _, tc := range []struct {
		name, document string
		want           RecoveryConfig
		bad            bool
	}{
		{"absent", `<html><script src="/boot.js"></script></html>`, DefaultRecoveryConfig(), false},
		{"external ignored", `<script src="/boot.js">globalThis["__DSH_CONNECTION_RECOVERY__"] = null;</script>`, DefaultRecoveryConfig(), false},
		{"partial", assignment(`{"backoffFactor":1.5,"backoffBaseMs":200}`), RecoveryConfig{200, 1.5, 10000, 3000, 15000}, false},
		{"last assignment resets defaults", assignment(`{"backoffBaseMs":20}`) + assignment(`{"backoffFactor":3}`), RecoveryConfig{500, 3, 10000, 3000, 15000}, false},
		{"unknown ignored", assignment(`{"future":null}`), DefaultRecoveryConfig(), false},
		{"null", assignment(`null`), RecoveryConfig{}, true},
		{"zero", assignment(`{"backoffBaseMs":0}`), RecoveryConfig{}, true},
		{"fractional timer", assignment(`{"generationReadyTimeoutMs":4.5}`), RecoveryConfig{}, true},
		{"string timer", assignment(`{"generationReadyWarnMs":"3000"}`), RecoveryConfig{}, true},
		{"overflow", assignment(`{"backoffMaxMs":2147483648}`), RecoveryConfig{}, true},
		{"null factor", assignment(`{"backoffFactor":null}`), RecoveryConfig{}, true},
		{"factor below one", assignment(`{"backoffFactor":0.5}`), RecoveryConfig{}, true},
		{"executable suffix", assignment(`{"backoffFactor":2};globalThis.pwned=true`), RecoveryConfig{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseConnectionRecovery(tc.document)
			if (err != nil) != tc.bad || !tc.bad && got != tc.want {
				t.Fatalf("got=%+v err=%v want=%+v bad=%v", got, err, tc.want, tc.bad)
			}
		})
	}
}

func TestConnectionRecoveryReadsAuthenticatedBoundedBootDocument(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		bad    bool
	}{
		{"policy", 200, `<script>globalThis["__DSH_CONNECTION_RECOVERY__"]={"backoffBaseMs":25}</script>`, false},
		{"status", 503, "synthetic-private-body", true},
		{"oversize", 200, strings.Repeat("x", 4*1024*1024+1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/" || r.Method != http.MethodGet || r.Header.Get("Cookie") != "fixture=ok" {
					t.Errorf("request %s %s cookie=%q", r.Method, r.URL.Path, r.Header.Get("Cookie"))
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			client := &Client{Endpoint: Endpoint{Origin: srv.URL, Cookie: "fixture=ok"}}
			got, err := client.ReadConnectionRecovery(context.Background())
			if (err != nil) != tc.bad {
				t.Fatalf("policy=%+v err=%v", got, err)
			}
			if err != nil && strings.Contains(err.Error(), tc.body) {
				t.Fatal("bootstrap response body escaped into diagnostic")
			}
			if !tc.bad && got.BackoffBaseMS != 25 {
				t.Fatal(got)
			}
		})
	}
}
