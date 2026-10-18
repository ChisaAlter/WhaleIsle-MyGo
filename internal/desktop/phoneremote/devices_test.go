package phoneremote

import (
	"reflect"
	"testing"

	"whaleisle/internal/credentials"
)

func TestDeviceNamePrefersName(t *testing.T) {
	got := DeviceName(map[string]any{
		"id":   "abcdef1234567890",
		"name": "iPhone",
	})
	if got != "iPhone" {
		t.Fatalf("DeviceName = %q", got)
	}
}

func TestDeviceNameFallsBackToID(t *testing.T) {
	got := DeviceName(map[string]any{"id": "dev-42"})
	if got != "dev-42" {
		t.Fatalf("DeviceName = %q", got)
	}
}

func TestDeviceNameEmpty(t *testing.T) {
	if DeviceName(nil) != "" {
		t.Fatal("expected empty for nil")
	}
	if DeviceName(map[string]any{}) != "" {
		t.Fatal("expected empty for empty map")
	}
}

func TestDeviceNames(t *testing.T) {
	devices := []credentials.RemoteDevice{
		{"name": "iPhone"},
		{"id": "pad-1"},
		{},
	}
	got := DeviceNames(devices)
	want := []string{"iPhone", "pad-1", "#3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DeviceNames = %v, want %v", got, want)
	}
}
