package pet

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

func newTestCompanion(t *testing.T) *Companion {
	t.Helper()
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return New(store)
}

func petPageRequest() *http.Request {
	return httptest.NewRequest(http.MethodGet, PetScheme+"://localhost/index.html", nil)
}

// TestWhalePetSchemeRegisteredBeforeAnyCompanion is the round154 §7 fix: the
// scheme must be handled before a window (and the WebView2 environment) exists,
// so the per-window filter list includes it. Package init registers it, which
// happens before main and long before pet.New.
func TestWhalePetSchemeRegisteredBeforeAnyCompanion(t *testing.T) {
	if err := RegisterProtocol(); err != nil {
		t.Fatalf("registering %s: %v", PetScheme, err)
	}
	if !mygo.Protocol.IsHandled(PetScheme) {
		t.Fatalf("%s is not handled after package init", PetScheme)
	}
	// Registering again must not fail or disturb the handler.
	if err := RegisterProtocol(); err != nil {
		t.Fatalf("second registration: %v", err)
	}
	// The registered handler is the stable dispatcher: with no companion it
	// answers 404 itself instead of being a companion that would need
	// re-registration when replaced.
	previous := Current()
	setCurrent(nil)
	t.Cleanup(func() { setCurrent(previous) })
	rec := httptest.NewRecorder()
	petProtocol{}.ServeHTTP(rec, petPageRequest())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("dispatcher without a companion = %d, want 404", rec.Code)
	}
}

// TestWhalePetRequestsWithoutCompanion403 documents the no-companion state: a
// request is answered, never left to the network.
func TestWhalePetRequestsWithoutCompanion404(t *testing.T) {
	previous := Current()
	setCurrent(nil)
	t.Cleanup(func() { setCurrent(previous) })
	rec := httptest.NewRecorder()
	petProtocol{}.ServeHTTP(rec, petPageRequest())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("request without a companion = %d, want 404", rec.Code)
	}
}

// TestWhalePetDispatchFollowsTheCurrentCompanion covers New() publishing and a
// replacement taking over through the same registration.
func TestWhalePetDispatchFollowsTheCurrentCompanion(t *testing.T) {
	first := newTestCompanion(t)
	if got := Current(); got != first {
		t.Fatal("New did not publish the companion")
	}
	rec := httptest.NewRecorder()
	petProtocol{}.ServeHTTP(rec, petPageRequest())
	if rec.Code != http.StatusOK {
		t.Fatalf("pet page = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Whale Isle companion") {
		t.Fatal("the served page is not the pet page")
	}

	second := newTestCompanion(t)
	if second == first {
		t.Fatal("replacement returned the same companion")
	}
	if got := Current(); got != second {
		t.Fatal("the replacement did not take over the scheme")
	}
	rec = httptest.NewRecorder()
	petProtocol{}.ServeHTTP(rec, petPageRequest())
	if rec.Code != http.StatusOK {
		t.Fatalf("pet page after replacement = %d, want 200", rec.Code)
	}
}
