//go:build windows && (amd64 || arm64)

package windows

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"net/url"
	"strings"

	"github.com/egoist/mygo/internal/platform"
)

// Windows limits Tag and Group to 64 characters on current systems (16 on
// older Windows 10). Fixed 16-character SHA256 keys work on both; the original
// ID lives in the launch XML and is never truncated.
func toastKey(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:8])
}

func toastLaunch(id string) string { return "mygo-notification:" + url.QueryEscape(id) }
func toastActionLaunch(id, action string) string {
	return toastLaunch(id) + "&action=" + url.QueryEscape(action)
}
func toastActivationParts(raw string) (string, string) {
	parts := strings.SplitN(raw, "&action=", 2)
	id := toastID(parts[0])
	action := ""
	if len(parts) == 2 {
		action, _ = url.QueryUnescape(parts[1])
	}
	return id, action
}

func toastID(raw string) string {
	if !strings.HasPrefix(raw, "mygo-notification:") {
		return ""
	}
	id, _ := url.QueryUnescape(strings.TrimPrefix(raw, "mygo-notification:"))
	return id
}

func toastXML(n *platform.Notification) string {
	t := n.Toast
	if t != nil && t.XML != "" {
		return toastCustomXML(n.ID, t.XML)
	}
	var b bytes.Buffer
	b.WriteString(`<toast launch="`)
	xml.EscapeText(&b, []byte(toastLaunch(n.ID)))
	b.WriteByte('"')
	if t != nil {
		if t.Duration != "" {
			b.WriteString(` duration="`)
			xml.EscapeText(&b, []byte(t.Duration))
			b.WriteByte('"')
		}
		if t.Scenario != "" {
			b.WriteString(` scenario="`)
			xml.EscapeText(&b, []byte(t.Scenario))
			b.WriteByte('"')
		}
	}
	b.WriteString(`><visual><binding template="ToastGeneric">`)
	for _, text := range []string{n.Title, n.Subtitle, n.Body} {
		if text == "" {
			continue
		}
		b.WriteString(`<text>`)
		xml.EscapeText(&b, []byte(text))
		b.WriteString(`</text>`)
	}
	if t != nil {
		for _, img := range []struct{ src, placement string }{{t.Icon, "appLogoOverride"}, {t.Hero, "hero"}} {
			if img.src == "" {
				continue
			}
			b.WriteString(`<image placement="` + img.placement + `" src="`)
			xml.EscapeText(&b, []byte(img.src))
			b.WriteString(`"/>`)
		}
	}
	b.WriteString(`</binding></visual>`)
	if t != nil && len(t.Actions) > 0 {
		b.WriteString(`<actions>`)
		for _, a := range t.Actions {
			b.WriteString(`<action activationType="foreground" content="`)
			xml.EscapeText(&b, []byte(a.Label))
			b.WriteString(`" arguments="`)
			xml.EscapeText(&b, []byte(toastActionLaunch(n.ID, a.Arguments)))
			b.WriteString(`"/>`)
		}
		b.WriteString(`</actions>`)
	}
	if n.Silent {
		b.WriteString(`<audio silent="true"/>`)
	}
	b.WriteString(`</toast>`)
	return b.String()
}

// Custom XML keeps the full Windows schema. Only foreground activation
// arguments receive the same ID envelope as generated templates, so cold and
// live action clicks carry identical user arguments and notification identity.
func toastCustomXML(id, raw string) string {
	d := xml.NewDecoder(strings.NewReader(raw))
	var b bytes.Buffer
	e := xml.NewEncoder(&b)
	for {
		token, err := d.Token()
		if err != nil {
			break
		}
		if start, ok := token.(xml.StartElement); ok && (start.Name.Local == "toast" || start.Name.Local == "action") {
			foreground := true
			key := "launch"
			if start.Name.Local == "action" {
				key = "arguments"
			}
			action := ""
			found := false
			for _, a := range start.Attr {
				if a.Name.Local == "activationType" && a.Value != "foreground" {
					foreground = false
				}
				if a.Name.Local == key {
					action = a.Value
				}
			}
			if foreground {
				for i, a := range start.Attr {
					if a.Name.Local == key {
						start.Attr[i].Value = toastActionLaunch(id, action)
						found = true
					}
				}
				if !found {
					start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: key}, Value: toastLaunch(id)})
				}
				token = start
			}
		}
		if err := e.EncodeToken(token); err != nil {
			return raw
		}
	}
	if err := e.Flush(); err != nil {
		return raw
	}
	return b.String()
}

func toastIDFromXML(raw string) string {
	d := xml.NewDecoder(strings.NewReader(raw))
	for {
		t, err := d.Token()
		if err != nil {
			return ""
		}
		if e, ok := t.(xml.StartElement); ok {
			if e.Name.Local != "toast" {
				return ""
			}
			for _, a := range e.Attr {
				if a.Name.Local == "launch" {
					id, _ := toastActivationParts(a.Value)
					return id
				}
			}
			return ""
		}
	}
}
