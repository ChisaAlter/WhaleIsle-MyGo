package frame

import (
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/sidebar"
)

// A store's remembered account is released when the shell retires the store,
// and releasing one store leaves every other store's memory alone.
func TestForgetExternalAccountReleasesOnlyThatStore(t *testing.T) {
	retired, kept, next := session.New(), session.New(), session.New()
	t.Cleanup(func() {
		ForgetExternalAccount(retired)
		ForgetExternalAccount(kept)
		ForgetExternalAccount(next)
	})

	RememberExternalAccount(retired, sidebar.Account{Name: "retired"})
	RememberExternalAccount(kept, sidebar.Account{Name: "kept"})

	ForgetExternalAccount(retired)
	if _, ok := RememberedExternalAccount(retired); ok {
		t.Fatal("retired store is still remembered")
	}
	if account, ok := RememberedExternalAccount(kept); !ok || account.Name != "kept" {
		t.Fatalf("forgetting one store dropped another: %#v, %v", account, ok)
	}

	RememberExternalAccount(next, sidebar.Account{Name: "next"})
	if account, ok := RememberedExternalAccount(next); !ok || account.Name != "next" {
		t.Fatalf("new store was not remembered: %#v, %v", account, ok)
	}
	if _, ok := RememberedExternalAccount(nil); ok {
		t.Fatal("a nil store is remembered")
	}
}
