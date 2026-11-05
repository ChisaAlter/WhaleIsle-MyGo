package pet

import (
	"encoding/json"
	"html"
	"strings"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/theme"
)

type petPresentation struct {
	Locale, Scheme, Font string
	Light, Dark          theme.Colors
}

func (p *Companion) presentation() petPresentation {
	cfg := config.Defaults()
	if p.Store != nil {
		cfg = p.Store.Get()
	}
	t := theme.Resolve(cfg.Theme)
	return petPresentation{Locale: cfg.Locale, Scheme: t.Scheme, Font: theme.RenderPreferences().Interface.Family, Light: t.Light, Dark: t.Dark}
}

func renderPresentationPage(page []byte, v petPresentation) []byte {
	b, _ := json.Marshal(v)
	return []byte(strings.ReplaceAll(string(page), "{{PRESENTATION}}", html.EscapeString(string(b))))
}
