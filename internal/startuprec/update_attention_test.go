package startuprec

import "testing"

// The fixture has no native APIs, network, process, download or installer hooks.
// Returning to confirmation is a separate action from resolving a prompt.
func TestUpdateAttentionLifecycle(t *testing.T) {
	base := UpdateAttentionScope{Window: "window-1", Host: "host-1", HostGeneration: 1, CatalogGeneration: 1, Route: "direct", Tag: "v2", Version: "2"}
	t.Run("background one reminder and click only returns to confirmation", func(t *testing.T) {
		var a UpdateAttention
		ticket, remind := a.Observe(base, true, false)
		if !remind || !a.Current(ticket) || a.Prompting() {
			t.Fatal("background update must be pending without starting confirmation")
		}
		for i := 0; i < 3; i++ {
			if got, again := a.Observe(base, true, false); got != ticket || again {
				t.Fatal("ignored notification repeated")
			}
		}
		if !a.Current(ticket) || a.Prompting() {
			t.Fatal("notification click must only retain pending confirmation")
		}
		if !a.BeginPrompt(ticket) || a.BeginPrompt(ticket) {
			t.Fatal("prompt must be single flight")
		}
		if !a.Resolve(ticket) || a.Current(ticket) || a.Resolve(ticket) {
			t.Fatal("resolved prompt must revoke its ticket once")
		}
		if _, again := a.Observe(base, true, false); again || a.Pending() {
			t.Fatal("Later/decline repeated the update")
		}
	})
	t.Run("focus and unavailable notification retain visible fallback", func(t *testing.T) {
		var a UpdateAttention
		ticket, remind := a.Observe(base, true, false)
		if !remind {
			t.Fatal("missing initial attention")
		}
		focused, again := a.Observe(base, true, true)
		if focused != ticket || again || !a.Pending() || !a.BeginPrompt(ticket) {
			t.Fatal("pending fallback was lost")
		}
		a.CancelPrompt(ticket)
		if !a.Pending() || a.Prompting() || !a.BeginPrompt(ticket) {
			t.Fatal("navigation before display consumed confirmation")
		}
	})
	t.Run("foreground does not notify later on blur", func(t *testing.T) {
		var a UpdateAttention
		if _, remind := a.Observe(base, true, true); remind {
			t.Fatal("foreground notification")
		}
		if _, remind := a.Observe(base, true, false); remind {
			t.Fatal("blur reissued reminder")
		}
	})
	t.Run("window host catalog route and release expire callbacks", func(t *testing.T) {
		changes := []func(*UpdateAttentionScope){
			func(s *UpdateAttentionScope) { s.Window = "window-2" },
			func(s *UpdateAttentionScope) { s.Host = "host-2" },
			func(s *UpdateAttentionScope) { s.HostGeneration++ },
			func(s *UpdateAttentionScope) { s.CatalogGeneration++ },
			func(s *UpdateAttentionScope) { s.Route = "mirror" },
			func(s *UpdateAttentionScope) { s.Tag = "v3"; s.Version = "3" },
		}
		for i, change := range changes {
			var a UpdateAttention
			old, _ := a.Observe(base, true, false)
			a.BeginPrompt(old)
			next := base
			change(&next)
			fresh, remind := a.Observe(next, true, false)
			if a.Current(old) || a.Resolve(old) || !a.Current(fresh) || a.Prompting() {
				t.Fatalf("change %d retained old authority", i)
			}
			if i < 4 && remind {
				t.Fatalf("same version re-notified for owner change %d", i)
			}
			if i >= 4 && !remind {
				t.Fatalf("new route/version did not receive attention %d", i)
			}
		}
	})
	t.Run("disable clear and explicit download invalidate", func(t *testing.T) {
		var a UpdateAttention
		old, _ := a.Observe(base, true, false)
		a.Observe(base, false, false)
		if a.Current(old) || a.Pending() {
			t.Fatal("disabled preference left authority")
		}
		if _, remind := a.Observe(base, true, false); remind || a.Pending() {
			t.Fatal("explicit ignore was not remembered")
		}
		next := base
		next.Tag = "v3"
		next.Version = "3"
		fresh, _ := a.Observe(next, true, false)
		a.Clear()
		if a.Current(fresh) || a.Pending() {
			t.Fatal("clear did not revoke callback")
		}
		fresh, remind := a.Observe(next, true, false)
		if remind || !a.Current(fresh) {
			t.Fatal("clear should preserve per-version reminder dedup")
		}
		a.DismissCurrent()
		if a.Pending() || a.Current(fresh) {
			t.Fatal("explicit download left update attention pending")
		}
		if _, remind := a.Observe(next, true, false); remind || a.Pending() {
			t.Fatal("explicit download re-prompted")
		}
	})
}
