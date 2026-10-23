package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

type accessiblePrimitiveClient struct {
	primitiveClient
	info TextInputAccessibility
}

func (c *accessiblePrimitiveClient) TextInputAccessibility() TextInputAccessibility { return c.info }

func TestAccessibleTextInputClientSnapshotAndReplacement(t *testing.T) {
	client := &accessiblePrimitiveClient{primitiveClient: primitiveClient{text: "A😀中文\nz", selection: TextInputSelection{Range: TextInputRange{Start: 3, End: 5}}}, info: TextInputAccessibility{Multiline: true, Placeholder: "Ask something"}}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreBox(c).Size(250, 70).Role(RoleTextField).Label("Custom draft").HandleTextInput(client)
			coreButton(c, "Other").AutoFocus()
		})
	}, 300, 150)
	defer tt.rt.close()
	if len(client.queries) != 0 {
		t.Fatal("accessibility queried the custom document before it was requested")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	field := accessNode(t, tt.h.access, platform.RoleTextField, "Custom draft")
	if field.Value != client.text || field.SelStart != 2 || field.SelEnd != 4 || field.Placeholder != "Ask something" || field.States&platform.AccessMultiline == 0 || field.Actions&platform.ActionSetValue == 0 || tt.h.access.Focus == field.ID {
		t.Fatalf("unfocused custom text snapshot: %+v", field)
	}
	// Accessible full replacement works when the window is not the active IME
	// target too. It must replace the document, rather than just the selection.
	tt.send(platform.SurfaceEvent{Kind: platform.SurfaceBlur})
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "新😀\n全文"})
	field = accessNode(t, tt.h.access, platform.RoleTextField, "Custom draft")
	if client.text != "新😀\n全文" || client.changes != 1 || field.Value != client.text || field.SelStart != 5 || field.SelEnd != 5 {
		t.Fatalf("whole-document replacement or selection: %q %+v", client.text, field)
	}
}

func TestAccessibleTextInputClientPoliciesAndLegacyClients(t *testing.T) {
	for _, policy := range []string{"legacy", "readonly", "password", "disabled"} {
		t.Run(policy, func(t *testing.T) {
			client := &accessiblePrimitiveClient{primitiveClient: primitiveClient{text: "secret😀", selection: TextInputSelection{Range: TextInputRange{End: 8}}}}
			client.info.ReadOnly = policy == "readonly"
			client.info.Password = policy == "password"
			var input TextInputClient = client
			if policy == "legacy" {
				input = &client.primitiveClient
			}
			tt := coreNewTester(func(c *context) {
				coreBox(c).Size(250, 70).Role(RoleTextField).Label("Custom").Value("explicit value").HandleTextInput(input).Disabled(policy == "disabled")
			}, 300, 100)
			defer tt.rt.close()
			tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
			field := accessNode(t, tt.h.access, platform.RoleTextField, "Custom")
			if policy == "legacy" && (field.Value != "explicit value" || len(client.queries) != 0 || field.Actions&platform.ActionSetValue != 0) {
				t.Fatal("legacy client opted into document queries/editing", field)
			}
			if policy == "password" && (field.Value != "" || len(client.queries) != 0 || field.States&platform.AccessPassword == 0 || field.SelStart != 0 || field.SelEnd != 0) {
				t.Fatal("password document leaked through the tree", field)
			}
			if policy == "readonly" && (field.Value != client.text || field.States&platform.AccessReadOnly == 0 || field.Actions&platform.ActionSetValue != 0) {
				t.Fatal("read-only metadata missing", field)
			}
			tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "replacement"})
			if policy == "password" {
				if client.text != "replacement" || client.changes != 1 || len(client.queries) != 0 {
					t.Fatal("password replacement queried/revealed its old document", client.text)
				}
			} else if client.changes != 0 {
				t.Fatal("read-only, disabled or unregistered client was edited")
			}
		})
	}
}

func TestAccessibleTextInputClientRejectsHiddenAndStaleTargets(t *testing.T) {
	for _, conceal := range []string{"invisible", "inert", "disabled-parent", "modal", "removed", "closed"} {
		t.Run(conceal, func(t *testing.T) {
			client := &accessiblePrimitiveClient{primitiveClient: primitiveClient{text: "keep"}}
			hide, dialog := false, false
			tt := coreNewTester(func(c *context) {
				coreColumn(c).Fill().Children(func() {
					if !hide || conceal != "removed" {
						parent := coreBox(c).Size(250, 70).Disabled(hide && conceal == "disabled-parent")
						if hide && conceal == "invisible" {
							parent.Invisible()
						}
						if hide && conceal == "inert" {
							parent.flags |= flagInert
						}
						parent.Children(func() { coreBox(c).Fill().Role(RoleTextField).Label("Custom").HandleTextInput(client) })
					}
					coreModal(c, &dialog, func() { coreButton(c, "Inside modal") })
				})
			}, 400, 250)
			defer tt.rt.close()
			tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
			field := accessNode(t, tt.h.access, platform.RoleTextField, "Custom")
			hide, dialog = true, conceal == "modal"
			tt.Frame()
			if conceal == "closed" {
				tt.rt.close()
			}
			tt.rt.accessAction(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "stale"})
			if client.changes != 0 || client.text != "keep" {
				t.Fatal("a cached inaccessible target was edited")
			}
		})
	}
}

// Model the native composition reset used by IMM32/GTK while retaining the
// ordinary headless host's event and paint behavior.
type accessCompositionHost struct {
	*headless
	composition platform.InputComposition
	resets      int
	onReset     func()
}

func (h *accessCompositionHost) setTextInput(input platform.TextInputState) {
	if h.ime.Active && h.ime.Client != input.Client {
		h.composition.Reset()
		h.resets++
	}
	h.headless.setTextInput(input)
	if !input.Active && h.onReset != nil {
		callback := h.onReset
		h.onReset = nil
		callback()
	}
}

func TestAccessibleTextInputClientReplacesCompositionAndResetsNativeState(t *testing.T) {
	client := &accessiblePrimitiveClient{primitiveClient: primitiveClient{text: "base", selection: TextInputSelection{Range: TextInputRange{Start: 4, End: 4}}}}
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(250, 70).Role(RoleTextField).Label("Custom").HandleTextInput(client).AutoFocus()
	}, 300, 100)
	defer tt.rt.close()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	field := accessNode(t, tt.h.access, platform.RoleTextField, "Custom")
	host := &accessCompositionHost{headless: tt.h}
	tt.rt.host = host
	tt.Compose("中文", 2)
	before := client.changes
	// The next event can arrive after native end-preedit. Full replacement must
	// restart native text services even though the same adapter retains focus.
	host.composition.End(tt.h.ime.Client)
	oldState := tt.h.ime
	tt.rt.accessAction(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "全文😀"})
	if client.text != "全文😀" || client.changes != before+1 || client.mark != nil || !tt.h.ime.Active || tt.h.ime.Client != oldState.Client {
		t.Fatal("replacement lost the client, doubled an edit or retained composition", client.text)
	}
	if host.resets != 1 {
		t.Fatal("accessible replacement did not reset native composition", host.resets)
	}
	host.composition.Replace(tt.h.ime.Client, nil, "!")
	tt.Frame()
	if client.text != "全文😀!" {
		t.Fatal("delayed native commit reused the old selection/marked range", client.text)
	}
}

func TestAccessibleTextInputClientRechecksAfterNativeReset(t *testing.T) {
	first := &accessiblePrimitiveClient{primitiveClient: primitiveClient{text: "first"}}
	second := &accessiblePrimitiveClient{primitiveClient: primitiveClient{text: "second"}}
	current := first
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(250, 70).Role(RoleTextField).Label("Custom").HandleTextInput(current).AutoFocus()
	}, 300, 100)
	defer tt.rt.close()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	field := accessNode(t, tt.h.access, platform.RoleTextField, "Custom")
	host := &accessCompositionHost{headless: tt.h}
	tt.rt.host = host
	host.onReset = func() {
		current = second
		tt.Frame()
	}
	tt.rt.accessAction(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "stale"})
	if first.text != "first" || second.text != "second" || first.changes != 0 || second.changes != 0 {
		t.Fatal("native cancellation redirected an old action to the replacement client")
	}
}

func TestAccessibleTextInputClientRejectsKeptRouterPage(t *testing.T) {
	client := &accessiblePrimitiveClient{primitiveClient: primitiveClient{text: "keep"}}
	router := NewRouter("/draft")
	router.Transition = TransitionNone
	tt := coreNewTester(func(c *context) {
		router.coreView(c, func(route *Route) {
			if route.Match("/draft") {
				coreBox(c).Size(250, 70).Role(RoleTextField).Label("Draft").HandleTextInput(client)
			} else {
				coreText(c, "Other page")
			}
		})
	}, 300, 100)
	defer tt.rt.close()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	field := accessNode(t, tt.h.access, platform.RoleTextField, "Draft")
	router.Push("/other")
	tt.Frame()
	if tt.rt.states[field.ID] == nil {
		t.Fatal("fixture did not retain the router page")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "stale"})
	if client.text != "keep" || client.changes != 0 {
		t.Fatal("a kept router page was edited through a stale accessibility node")
	}
}
