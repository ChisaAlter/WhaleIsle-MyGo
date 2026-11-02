package pluginslots

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
)

func TestGatewayExtensionOnlyReceivesAuthenticatedPrivatePaths(t *testing.T) {
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "upstream") }))
	defer host.Close()
	calls := 0
	g, err := NewGatewayWithHandler(host.URL, "fixture", func(w http.ResponseWriter, r *http.Request) bool {
		calls++
		io.WriteString(w, "extension")
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	entry, _ := g.EntryURL(host.URL + "/")
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	response, err := client.Get(entry)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	base := response.Request.URL.Scheme + "://" + response.Request.URL.Host
	plain, _ := http.Get(base + "/__whale_plugin_ui/probe")
	plain.Body.Close()
	if plain.StatusCode != http.StatusForbidden || calls != 0 {
		t.Fatal("extension ran before cookie authentication")
	}
	req, _ := http.NewRequest("GET", base+"/__whale_plugin_ui/probe", nil)
	req.Header.Set("Origin", "https://foreign.invalid")
	denied, _ := client.Do(req)
	denied.Body.Close()
	if denied.StatusCode != http.StatusForbidden || calls != 0 {
		t.Fatal("extension ran before origin validation")
	}
	for _, path := range []string{"/ordinary", "/__whale_keyboard", "/__whale_client_modules"} {
		res, e := client.Get(base + path)
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
	}
	if calls != 0 {
		t.Fatal("extension intercepted non-private or reserved route")
	}
	res, err := client.Get(base + "/__whale_plugin_ui/probe")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if calls != 1 || string(body) != "extension" {
		t.Fatalf("private handler did not receive accepted request %d %q", calls, body)
	}
}
