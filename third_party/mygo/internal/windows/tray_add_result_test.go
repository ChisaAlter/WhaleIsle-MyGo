package windows

import "testing"

// Synthetic Win32 BOOL results only: never loads a DLL or creates a real tray.
func TestTrayAddResultKeepsFailedIconUnregistered(t *testing.T) {
	added, err := trayAddResult(0)
	if err == nil || added {
		t.Fatal("failed NIM_ADD must return an error and remain unregistered")
	}
	for _, result := range []uintptr{1, 2} {
		added, err = trayAddResult(result)
		if err != nil || !added {
			t.Fatalf("nonzero BOOL %d is success: %v", result, err)
		}
	}
}
