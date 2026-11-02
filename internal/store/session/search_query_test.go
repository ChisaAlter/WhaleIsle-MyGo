package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"
	"unicode/utf16"

	"whaleisle/internal/dsh"
)

func TestSessionSearchNormalizationBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, input, sanitized, canonical string }{
		{"spaces", " foo bar \t", " foo bar \t", "foo bar"},
		{"nul", "\x00中\x00文\x00", "中文", "中文"},
		{"empty", " \x00\t", " \t", ""},
		{"chinese500", strings.Repeat("中", 500) + "文", strings.Repeat("中", 500), strings.Repeat("中", 500)},
		{"emojiFits", strings.Repeat("中", 498) + "😀文", strings.Repeat("中", 498) + "😀", strings.Repeat("中", 498) + "😀"},
		{"emojiCrosses", strings.Repeat("中", 499) + "😀文", strings.Repeat("中", 499), strings.Repeat("中", 499)},
		{"nulBeforeBound", strings.Repeat("a", 499) + "\x00中x", strings.Repeat("a", 499) + "中", strings.Repeat("a", 499) + "中"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeSessionSearchInput(tc.input); got != tc.sanitized {
				t.Fatalf("sanitized %q, want %q", got, tc.sanitized)
			}
			got := NormalizeSessionSearchQuery(tc.input)
			if got != tc.canonical || len(utf16.Encode([]rune(got))) > 500 || NewSessionBatchScope(true, "workspace", "default", tc.input).Query != got {
				t.Fatalf("canonical input/scope differs: %q want %q", got, tc.canonical)
			}
		})
	}
}

func TestSessionSearchCanonicalRPCScopeAndDebounce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queries := make(chan string, 4)
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://search-fixture.invalid"}, HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
			var body struct {
				RPCID   string `json:"rpcId"`
				Payload struct {
					Args struct {
						Request struct {
							Query string `json:"query"`
						} `json:"request"`
					} `json:"args"`
				} `json:"payload"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			queries <- body.Payload.Args.Request.Query
			response := httptest.NewRecorder()
			rpcOK(response, body.RPCID, map[string]any{"items": []any{}, "hasMore": false})
			return response.Result(), nil
		})}}
		st := New()
		st.Attach(client, nil)
		defer st.Detach()
		prefix := strings.Repeat("中", 498) + "😀"
		first, second := prefix+strings.Repeat("甲", 1000), prefix+strings.Repeat("乙", 1000)
		st.UpdateSessionSearch(context.Background(), first)
		st.mu.Lock()
		seq, timer := st.searchSeq, st.searchDebounce
		st.mu.Unlock()
		st.UpdateSessionSearch(context.Background(), second)
		st.mu.Lock()
		current, stable := st.searchInput, st.searchSeq == seq && st.searchDebounce == timer
		st.mu.Unlock()
		if current != prefix || !stable || NewSessionBatchScope(true, "workspace", "default", second).Query != current {
			t.Fatal("different suffix beyond 500 changed the canonical input, timer or scope")
		}
		time.Sleep(249 * time.Millisecond)
		synctest.Wait()
		select {
		case <-queries:
			t.Fatal("RPC preceded installed 250ms debounce")
		default:
		}
		time.Sleep(time.Millisecond)
		synctest.Wait()
		select {
		case got := <-queries:
			if got != prefix {
				t.Fatalf("RPC differs from scope: %q", got)
			}
		default:
			t.Fatal("no RPC at 250ms")
		}
		if got := st.Snapshot().SearchRPCQuery; got != prefix {
			t.Fatalf("published query %q", got)
		}
		st.SearchSessions(context.Background(), "\x00"+second)
		synctest.Wait()
		select {
		case got := <-queries:
			if got != prefix {
				t.Fatal("manual refresh lost normalization")
			}
		default:
			t.Fatal("manual refresh not dispatched")
		}
	})
}
