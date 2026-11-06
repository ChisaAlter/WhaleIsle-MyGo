package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestViewPR_missingGH(t *testing.T) {
	st := viewPR(context.Background(), t.TempDir(), func(string) (string, error) {
		return "", errors.New("not found")
	}, combinedExecRunner)
	if st.OK || st.Err != MsgMissingGH {
		t.Fatalf("%+v", st)
	}
}

func TestCreatePRDraft_missingGH(t *testing.T) {
	st := createPRDraft(context.Background(), t.TempDir(), func(string) (string, error) {
		return "", errors.New("not found")
	}, combinedExecRunner)
	if st.OK || st.Err != MsgMissingGH {
		t.Fatalf("%+v", st)
	}
}

func TestFirstHTTPURL(t *testing.T) {
	if got := firstHTTPURL("ok\nhttps://example.com/pr/1\n"); got != "https://example.com/pr/1" {
		t.Fatalf("got %q", got)
	}
}

func TestParsePRStatusJSON_currentAndLists(t *testing.T) {
	raw := []byte(`{
		"currentBranch":{"number":12,"title":"Fix login","url":"https://example.com/p/12","state":"OPEN"},
		"createdBy":[{"number":9,"title":"Mine","url":"https://example.com/p/9","state":"OPEN"}],
		"needsReview":[{"number":8,"title":"Review me","url":"https://example.com/p/8","state":"OPEN"}]
	}`)
	cur, created, review, err := ParsePRStatusJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cur == nil || cur.Number != 12 || cur.Title != "Fix login" || cur.State != "OPEN" {
		t.Fatalf("current = %+v", cur)
	}
	if len(created) != 1 || created[0].Number != 9 {
		t.Fatalf("createdBy = %+v", created)
	}
	if len(review) != 1 || review[0].Number != 8 {
		t.Fatalf("needsReview = %+v", review)
	}
}

func TestParsePRStatusJSON_nullCurrent(t *testing.T) {
	raw := []byte(`{"createdBy":[],"needsReview":[]}`)
	cur, created, review, err := ParsePRStatusJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cur != nil {
		t.Fatalf("current = %+v", cur)
	}
	if len(created) != 0 || len(review) != 0 {
		t.Fatalf("lists = %v %v", created, review)
	}
}

func TestParsePRListJSON(t *testing.T) {
	raw := []byte(`[
		{"number":3,"title":"A","url":"https://example.com/p/3","state":"OPEN"},
		{"number":2,"title":"B","url":"https://example.com/p/2","state":"MERGED"}
	]`)
	list, err := ParsePRListJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Number != 3 || list[1].State != "MERGED" {
		t.Fatalf("%+v", list)
	}
}

func TestFormatPRInfo(t *testing.T) {
	got := FormatPRInfo(PRInfo{Number: 12, Title: "Fix login", State: "OPEN"})
	if got != "#12 Fix login [OPEN]" {
		t.Fatalf("got %q", got)
	}
}

func TestPRStatusReport_fakeRunner(t *testing.T) {
	root := t.TempDir()
	var calls []string
	run := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		if name != "gh" {
			t.Fatalf("name = %q", name)
		}
		if dir != root {
			t.Fatalf("dir = %q", dir)
		}
		key := strings.Join(args, " ")
		calls = append(calls, key)
		switch {
		case strings.HasPrefix(key, "pr status"):
			return []byte(`{"currentBranch":{"number":7,"title":"Ship","url":"https://example.com/p/7","state":"OPEN"},"createdBy":[],"needsReview":[]}`), nil
		case strings.HasPrefix(key, "pr list"):
			if !strings.Contains(key, "--limit 5") {
				t.Fatalf("list args = %v", args)
			}
			return []byte(`[{"number":7,"title":"Ship","url":"https://example.com/p/7","state":"OPEN"}]`), nil
		default:
			return nil, fmt.Errorf("unexpected %v", args)
		}
	}
	rep := prStatusReport(context.Background(), root, 5, func(string) (string, error) {
		return "gh", nil
	}, run)
	if !rep.OK || rep.MissingGH || rep.Err != "" {
		t.Fatalf("%+v", rep)
	}
	if rep.Current == nil || rep.Current.Number != 7 {
		t.Fatalf("current = %+v", rep.Current)
	}
	if len(rep.Recent) != 1 || rep.Recent[0].Title != "Ship" {
		t.Fatalf("recent = %+v", rep.Recent)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %v", calls)
	}
}

func TestPRStatusReport_missingGH(t *testing.T) {
	rep := prStatusReport(context.Background(), t.TempDir(), 5, func(string) (string, error) {
		return "", errors.New("not found")
	}, combinedExecRunner)
	if rep.OK || !rep.MissingGH || rep.Err != MsgMissingGH {
		t.Fatalf("%+v", rep)
	}
}

func TestCheckoutPR_fakeRunner(t *testing.T) {
	root := t.TempDir()
	var seen []string
	run := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		seen = append([]string(nil), args...)
		return []byte("Switched to branch fix\n"), nil
	}
	st := checkoutPR(context.Background(), root, 42, func(string) (string, error) {
		return "gh", nil
	}, run)
	if !st.OK || st.Err != "" {
		t.Fatalf("%+v", st)
	}
	if len(seen) != 3 || seen[0] != "pr" || seen[1] != "checkout" || seen[2] != "42" {
		t.Fatalf("args = %v", seen)
	}
}

func TestViewPR_fakeRunnerJSON(t *testing.T) {
	root := t.TempDir()
	run := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		if args[0] == "repo" {
			return []byte(`{"nameWithOwner":"example/project","url":"https://github.com/example/project","defaultBranchRef":{"name":"main"}}`), nil
		}
		if got := strings.Join(args, " "); got != "pr view --json number,title,url,state" {
			t.Fatalf("current-branch PR lookup must preserve gh branch resolution: %s", got)
		}
		return []byte(`{"number":1,"title":"Hi","url":"https://github.com/example/project/pull/1","state":"OPEN"}`), nil
	}
	st := viewPR(context.Background(), root, func(string) (string, error) {
		return "gh", nil
	}, run)
	if !st.OK || st.URL != "https://github.com/example/project/pull/1" || st.Summary != "#1 Hi [OPEN]" {
		t.Fatalf("%+v", st)
	}
}
