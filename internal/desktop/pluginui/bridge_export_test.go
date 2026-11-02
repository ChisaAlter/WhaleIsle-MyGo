package pluginui

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func SessionRendererResponseFixture(owner *Owner, response *http.Response) error {
	if response.Request.Header.Get("X-Whale-Test-Unknown-Onboarding") == "1" && response.StatusCode == http.StatusOK && strings.HasPrefix(response.Request.URL.Path, "/plugins/") && strings.Contains(response.Header.Get("Content-Type"), "javascript") {
		// Make only the Models policy unknown; production filtering must detect
		// it and retire this view without executing the welcome module.
		raw, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		response.Body.Close()
		var decoder io.ReadCloser
		switch response.Header.Get("Content-Encoding") {
		case "gzip":
			decoder, err = gzip.NewReader(bytes.NewReader(raw))
		case "deflate":
			decoder, err = zlib.NewReader(bytes.NewReader(raw))
		}
		if err != nil {
			return err
		}
		if decoder != nil {
			raw, err = io.ReadAll(decoder)
			decoder.Close()
			if err != nil {
				return err
			}
			response.Header.Del("Content-Encoding")
		}
		raw = bytes.ReplaceAll(raw, []byte(`const credentialOnboarding = Config(payload === void 0 ? {} : payload).credentialOnboarding`), []byte(`const credentialOnboarding = false && Config(payload === void 0 ? {} : payload).credentialOnboarding`))
		response.Body = io.NopCloser(bytes.NewReader(raw))
		response.ContentLength = int64(len(raw))
		response.Header.Set("Content-Length", strconv.Itoa(len(raw)))
	}
	err := sessionResponse(response, response.Request.Header.Get("X-Whale-Test-No-Entry-Observation") != "1")
	owner.acceptSessionResponse(response)
	return err
}

// NewBridgeFixture lets the external functional suite use the production parser
// and lifecycle without a native window or a second production constructor.
func NewBridgeFixture() (*Owner, func(http.ResponseWriter, *http.Request) bool) {
	o := &Owner{selected: Subject{Kind: "list"}}
	return o, o.serve
}
