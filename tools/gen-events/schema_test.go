package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMiniSchema(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "mini-schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	inv, err := parseInventory(data)
	if err != nil {
		t.Fatalf("parseInventory: %v", err)
	}
	if inv.FormatVersion != 2 {
		t.Fatalf("formatVersion = %d", inv.FormatVersion)
	}
	if len(inv.Roots) != 3 {
		t.Fatalf("roots = %d, want 3", len(inv.Roots))
	}
	if inv.Roots[0].Kind != "header" || inv.Roots[0].Key != "SessionHeader" {
		t.Fatalf("root0 = %+v", inv.Roots[0])
	}
	if inv.Roots[1].Event != "turn/start" || inv.Roots[2].Event != "tool/call" {
		t.Fatalf("events = %q, %q", inv.Roots[1].Event, inv.Roots[2].Event)
	}
}

func TestParseRealPersistenceSchema(t *testing.T) {
	path := filepath.Join("..", "..", "runtime", "vendor", "deepseek-harness", "docs", "persistence-schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		path = filepath.Join("..", "..", "reference", "electron", "vendor", "deepseek-harness", "docs", "persistence-schema.json")
		data, err = os.ReadFile(path)
		if err != nil {
			t.Skipf("persistence-schema.json not available: %v", err)
		}
	}
	inv, err := parseInventory(data)
	if err != nil {
		t.Fatalf("parseInventory(%s): %v", path, err)
	}
	if inv.FormatVersion != 2 {
		t.Fatalf("formatVersion = %d", inv.FormatVersion)
	}
	if len(inv.Roots) < 50 {
		t.Fatalf("roots = %d, want >= 50", len(inv.Roots))
	}
	nEvent := 0
	for _, r := range inv.Roots {
		if r.Kind == "event" {
			nEvent++
		}
	}
	if nEvent < 40 {
		t.Fatalf("events = %d, want >= 40", nEvent)
	}
	t.Logf("parsed %s: %d roots, %d events", path, len(inv.Roots), nEvent)
}

func TestParseRejectsBadVersion(t *testing.T) {
	_, err := parseInventory([]byte(`{"formatVersion":99,"roots":[{"key":"X","kind":"header","digest":"d","schema":{"root":0,"nodes":[{"kind":"primitive","type":"string"}]}}]}`))
	if err == nil {
		t.Fatal("expected error for formatVersion 99")
	}
}
