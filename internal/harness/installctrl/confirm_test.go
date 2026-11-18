package installctrl

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// The plugin fails closed on anything but {ok:true}, so the route has to answer
// with a refusal — never an error shape the plugin might read as consent.
func TestConfirmRouteForwardsTheDecisionAndFailsClosed(t *testing.T) {
	var seen ConfirmRequest
	srv := New(Options{Desktop: DesktopOps{
		Confirm: func(_ context.Context, ask ConfirmRequest) (any, error) {
			seen = ask
			return map[string]any{"ok": ask.Tool == "deepseek_tts"}, nil
		},
	}})
	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()
	call := func(target string, ops DesktopOps, tool string) map[string]any {
		t.Helper()
		body, _ := json.Marshal(ConfirmRequest{Tool: tool, Action: "request", Effect: "external", Summary: "把文本上传到付费接口", Details: "计费字符数 12"})
		req, err := http.NewRequest(http.MethodPost, srv.URL()+"/desktop/confirm", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+srv.Token())
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	allowed := call("deepseek_tts", srv.opts.Desktop, "deepseek_tts")
	if allowed["ok"] != true {
		t.Fatalf("allowed answer = %#v", allowed)
	}
	if seen.Tool != "deepseek_tts" || seen.Summary == "" || seen.Details == "" || seen.Effect != "external" {
		t.Fatalf("request did not reach the desktop: %+v", seen)
	}

	refused := call("deepseek_ocr", srv.opts.Desktop, "deepseek_ocr")
	if refused["ok"] != false {
		t.Fatalf("refusal answer = %#v", refused)
	}

	bare := New(Options{})
	if err := bare.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer bare.Stop()
	req, err := http.NewRequest(http.MethodPost, bare.URL()+"/desktop/confirm", bytes.NewReader([]byte(`{"tool":"deepseek_asr"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+bare.Token())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out["ok"] != false {
		t.Fatalf("a desktop without the operation must refuse, got %#v", out)
	}
}
