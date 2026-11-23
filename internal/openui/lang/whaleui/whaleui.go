// Package whaleui provides the Whale UI component library for the native
// OpenUI Lang runtime: the generated library definition (library.json, see
// gen/gen.mjs), the library function implementations and the genui option
// schemas of the specialized components.
package whaleui

//go:generate node gen/gen.mjs

import (
	_ "embed"
	"sync"

	"whaleisle/internal/openui/lang"
)

//go:embed library.json
var libraryJSON []byte

// LibraryJSON returns the generated library document.
func LibraryJSON() []byte { return libraryJSON }

var (
	once    sync.Once
	lib     *lang.Library
	genui   *Genui
	loadErr error
)

func load() {
	lib, loadErr = lang.LoadLibrarySpec(libraryJSON, Functions(), lang.WhaleProfile)
	if loadErr != nil {
		return
	}
	// Whale's Form has no name prop: the form element's identity names it.
	lib.FormComponents = map[string]string{"Form": ""}
	genui, loadErr = newGenui(lib.Extra["genui"])
	if loadErr != nil {
		return
	}
	lib.ValidateElement = genui.ValidateElement
}

// Library returns the shared Whale UI library. It is read-only.
func Library() (*lang.Library, error) {
	once.Do(load)
	return lib, loadErr
}

// GenuiValidator returns the specialized-component option validator.
func GenuiValidator() (*Genui, error) {
	once.Do(load)
	return genui, loadErr
}

// Languages are the program language tags the runtime accepts.
var Languages = []string{"whale-ui", "openui", "openui-lang", "dsh-ui"}

// ValidLanguage reports a supported language tag ("" means whale-ui).
func ValidLanguage(tag string) bool {
	if tag == "" {
		return true
	}
	for _, l := range Languages {
		if l == tag {
			return true
		}
	}
	return false
}
