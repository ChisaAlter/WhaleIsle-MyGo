package composer

import "testing"

func TestModelPickerKeepsInputAndRollsBackUncommittedChoice(t *testing.T) {
	var p modelPickerChoice
	p.sync("a", "saved")
	p.Value = "picked"
	p.sync("a", "saved")
	if got := p.take(); got != "picked" {
		t.Fatalf("input lost: %q", got)
	}
	if p.Value != "saved" {
		t.Fatalf("unconfirmed selection painted: %q", p.Value)
	}
	p.Value = "late"
	p.sync("b", "")
	if p.Value != "" {
		t.Fatal("previous session choice leaked")
	}
	p.sync("b", "confirmed")
	if p.Value != "confirmed" {
		t.Fatal("Host confirmation was not displayed")
	}
}
