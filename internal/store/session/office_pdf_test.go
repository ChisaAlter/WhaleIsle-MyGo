package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
)

func TestRenderOfficePDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/api/officeToPdf/render" {
			// Select may kick off follow/goals/skills; ignore.
			rpcOK(w, body["rpcId"], map[string]any{})
			return
		}
		rpcOK(w, body["rpcId"], map[string]any{
			"missingFonts": []string{},
			"generation":   "g1",
			"offset":       0,
			"data":         nil,
			"eof":          true,
			"absolutePath": `C:\tmp\out.pdf`,
			"version":      "1",
		})
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.Select("sess-1")
	abs, err := st.RenderOfficePDF(context.Background(), "doc.docx")
	if err != nil {
		t.Fatal(err)
	}
	if abs != `C:\tmp\out.pdf` {
		t.Fatalf("%q", abs)
	}
}
