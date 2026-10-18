package settings

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/providerdiscovery"
)

func TestSP23HostDiscoveryAvailableWithoutEndpoint(t *testing.T) {
	for _, protocol := range []string{"", "google-generative-ai"} {
		t.Run(protocol, func(t *testing.T) {
			calls := 0
			row := providerdiscovery.Row{ID: "native", Name: "Native", Provider: "native", Namespace: "llm-pi-ai", Protocol: protocol, Writable: true, CredentialKnown: true, CredentialConfigured: true}
			a := &ProviderDiscoveryAccess{
				Snapshot: providerdiscovery.Snapshot{Connected: true, Ready: true, Ticket: 11, SelectedID: row.ID, Rows: []providerdiscovery.Row{row}, Protocol: protocol},
				OnDiscover: func(ticket uint64) {
					if ticket != 11 {
						t.Fatalf("wrong action ticket: %d", ticket)
					}
					calls++
				},
			}
			tt := ui.NewTester(func(c *ui.Context) { providerDiscoveryPane(c, a, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150)) }, 1200, 1800)
			text := strings.Join(tt.Texts(), "\n")
			if !strings.Contains(text, "Host · llm-pi-ai · native") || strings.Contains(text, "GET ") || strings.Contains(text, "this probe does not use it") {
				t.Fatalf("wrong Host discovery presentation: %s", text)
			}
			if err := tt.Click(providerCopy("通过 Host 探测模型", "Discover models through Host")); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatal("missing endpoint or native protocol blocked Host discovery")
			}
		})
	}
}
