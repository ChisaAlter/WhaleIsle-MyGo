//go:build windows && (amd64 || arm64)

// A windowsgui fixture COM server, launched only by the opt-in native test with
// a unique CLSID/AppID and an output inside that test's temporary directory.
package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/egoist/mygo/internal/windows"
)

func main() {
	// COM appends -Embedding to LocalServer32's explicit fixture arguments.
	if len(os.Args) < 4 {
		return
	}
	for _, arg := range os.Args[4:] {
		if arg != "-Embedding" && arg != "/Embedding" {
			return
		}
	}
	b := windows.New()
	message, err := b.ReceiveNotificationActivation(os.Args[1], os.Args[2], 15*time.Second)
	result := struct {
		AppID, Arguments string
		UserInput        map[string]string
		Error            string
	}{AppID: message.AppID, Arguments: message.Arguments, UserInput: message.UserInput}
	if err != nil {
		result.Error = err.Error()
	}
	raw, _ := json.Marshal(result)
	_ = os.WriteFile(os.Args[3], raw, 0600)
}
