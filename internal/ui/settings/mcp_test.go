package settings

import "testing"

func TestFormatMCPStatus(t *testing.T) {
	if got := FormatMCPStatus("", "", nil); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := FormatMCPStatus("ready", "ok", nil); got != "ready / ok" {
		t.Fatalf("got %q", got)
	}
	err := "boom"
	if got := FormatMCPStatus("failed", "down", &err); got != "failed / down / boom" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatMCPServerCount(t *testing.T) {
	if FormatMCPServerCount(0) != "0" || FormatMCPServerCount(3) != "3" {
		t.Fatal("count format")
	}
}

func TestMCPRowDetailParts(t *testing.T) {
	row := MCPServerRow{ID: "local", Origin: "host", Enabled: true, Status: "ready / ok"}
	if row.ID == "" || !row.Enabled {
		t.Fatal("row fields")
	}
}
