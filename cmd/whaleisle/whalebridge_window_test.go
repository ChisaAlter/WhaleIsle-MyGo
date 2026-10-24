package main

import "testing"

// TestSameManagementOrigin keeps the embedded window on the instance's loopback
// origin: the validated management address never authorizes another host, and a
// path or query on the same origin stays allowed.
func TestSameManagementOrigin(t *testing.T) {
	address := "http://127.0.0.1:8765/?k=0123456789abcdef0123456789abcdef"
	allowed := []string{
		address,
		"http://127.0.0.1:8765/",
		"http://127.0.0.1:8765/settings?k=0123456789abcdef0123456789abcdef",
		"HTTP://127.0.0.1:8765/anything",
	}
	for _, target := range allowed {
		if !sameManagementOrigin(target, address) {
			t.Fatalf("blocked same-origin target %q", target)
		}
	}
	blocked := []string{
		"",
		"http://127.0.0.1:9999/",
		"https://127.0.0.1:8765/",
		"https://example.com/",
		"file:///etc/passwd",
		"javascript:alert(1)",
	}
	for _, target := range blocked {
		if sameManagementOrigin(target, address) {
			t.Fatalf("allowed foreign target %q", target)
		}
	}
	if sameManagementOrigin(address, "not a url") {
		t.Fatal("allowed navigation with an unparsable address")
	}
}
