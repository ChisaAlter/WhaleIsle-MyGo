package dsh

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
)

func TestCallMultipartAssemblesBytes(t *testing.T) {
	payload := []byte{0, 128, 255}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcID, _ := body["rpcId"].(string)
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		meta := map[string]any{
			"type":  "server-response",
			"rpcId": rpcID,
			"result": map[string]any{
				"ok":    true,
				"value": map[string]any{"offset": 7, "data": nil, "eof": true},
			},
			"attachments": []map[string]any{
				{"path": []any{"data"}, "codec": "bytes", "part": "bytes-0"},
			},
		}
		writePart(t, mw, "metadata", "application/json", mustJSON(meta))
		writePart(t, mw, "bytes-0", "application/octet-stream", payload)
		_ = mw.Close()
		w.Header().Set("Content-Type", mw.FormDataContentType())
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	c := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	mr, err := c.CallMultipart(context.Background(), "workspaceFiles/readBytes", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if got := mr.Parts["bytes-0"]; string(got) != string(payload) {
		t.Fatalf("parts = %v", mr.Parts)
	}
	var out struct {
		Offset float64 `json:"offset"`
		Data   []byte  `json:"data"`
		Eof    bool    `json:"eof"`
	}
	if err := json.Unmarshal(mr.Value, &out); err != nil {
		t.Fatal(err)
	}
	if out.Offset != 7 || !out.Eof || string(out.Data) != string(payload) {
		t.Fatalf("value = %#v", out)
	}
}

func TestCallMultipartNestedPath(t *testing.T) {
	payload := []byte("pdf")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcID, _ := body["rpcId"].(string)
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		meta := map[string]any{
			"type":  "server-response",
			"rpcId": rpcID,
			"result": map[string]any{
				"ok": true,
				"value": map[string]any{
					"metadata": map[string]any{"attachments": map[string]any{"content": nil}},
				},
			},
			"attachments": []map[string]any{
				{"path": []any{"metadata", "attachments", "content"}, "codec": "bytes", "part": "bytes-0"},
			},
		}
		writePart(t, mw, "metadata", "application/json", mustJSON(meta))
		writePart(t, mw, "bytes-0", "application/octet-stream", payload)
		_ = mw.Close()
		w.Header().Set("Content-Type", mw.FormDataContentType())
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	c := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	mr, err := c.CallMultipart(context.Background(), "officeToPdf/render", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Metadata struct {
			Attachments struct {
				Content []byte `json:"content"`
			} `json:"attachments"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(mr.Value, &out); err != nil {
		t.Fatal(err)
	}
	if string(out.Metadata.Attachments.Content) != string(payload) {
		t.Fatalf("nested = %#v", out)
	}
}

func writePart(t *testing.T, mw *multipart.Writer, name, ct string, body []byte) {
	t.Helper()
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="`+name+`"`)
	if ct != "" {
		h.Set("Content-Type", ct)
	}
	p, err := mw.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write(body); err != nil {
		t.Fatal(err)
	}
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
