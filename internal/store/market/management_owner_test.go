package market

import (
	"path/filepath"
	"testing"

	"whaleisle/internal/dsh"
)

func TestConfirmationIsRetiredByEveryRuntimeAttachment(t *testing.T) {
	for _, change := range []string{"replacement", "same-client", "detach", "A-B-A"} {
		t.Run(change, func(t *testing.T) {
			s := NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
			client := &dsh.Client{}
			s.Attach(client, nil)
			runs := 0
			s.RequestConfirmation("Remove plugin from original Host", func() { runs++ })
			switch change {
			case "replacement":
				s.Attach(&dsh.Client{}, nil)
			case "same-client":
				s.Attach(client, nil)
			case "detach":
				s.Detach()
			case "A-B-A":
				s.Attach(&dsh.Client{}, nil)
				s.Attach(client, nil)
			}
			if s.Confirmation() != "" {
				t.Fatal("replacement retained the original Host's confirmation")
			}
			s.ResolveConfirmation(true)
			if runs != 0 {
				t.Fatal("replacement executed an original Host action")
			}
		})
	}
}

func TestConfirmationRunsOnceOnlyInItsCurrentRuntime(t *testing.T) {
	s := NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	s.Attach(&dsh.Client{}, nil)
	runs := 0
	s.RequestConfirmation("Install fixture", func() { runs++ })
	s.ResolveConfirmation(true)
	s.ResolveConfirmation(true)
	if runs != 1 || s.Confirmation() != "" {
		t.Fatal("current confirmation did not execute exactly once")
	}
	s.RequestConfirmation("Cancel fixture", func() { runs++ })
	s.ResolveConfirmation(false)
	s.ResolveConfirmation(true)
	if runs != 1 {
		t.Fatal("cancelled confirmation executed")
	}
}
