package harness

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"
)

//go:embed proactive-readonly.mjs
var proactiveReadOnlyModule string

//go:embed proactive-tools.mjs
var proactiveToolsModule string

//go:embed proactive-delivery.mjs
var proactiveDeliveryModule string

//go:embed proactive-provenance.mjs
var proactiveProvenanceModule string

func writeProactiveOverlay(o PrepareOptions) (string, error) {
	entry, err := writeRawOverlay(o.ProfileDir, "mygo-proactive", "index.mjs", proactiveReadOnlyModule)
	if err != nil {
		return "", err
	}
	if _, err := writeRawOverlay(o.ProfileDir, "mygo-proactive", "tools.mjs", proactiveToolsModule); err != nil {
		return "", err
	}
	if _, err := writeRawOverlay(o.ProfileDir, "mygo-proactive", "proactive-provenance.mjs", proactiveProvenanceModule); err != nil {
		return "", err
	}
	if _, err := writeRawOverlay(o.ProfileDir, "mygo-proactive", "delivery.mjs", proactiveDeliveryModule); err != nil {
		return "", err
	}
	manifest := `{"name":"mygo-proactive-readonly","version":"1.0.0","type":"module","main":"index.mjs"}`
	if _, err := writeRawOverlay(o.ProfileDir, "mygo-proactive", "package.json", manifest); err != nil {
		return "", err
	}
	entry, err = filepath.Abs(entry)
	if err != nil {
		return "", err
	}
	path := filepath.ToSlash(entry)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	module, _ := json.Marshal((&url.URL{Scheme: "file", Path: path}).String())
	return writeRawOverlay(o.ProfileDir, "mygo-proactive", "desktop-proactive.patch.yml", "- insert:\n    - id: mygo-proactive-readonly\n      name: "+string(module)+"\n")
}
