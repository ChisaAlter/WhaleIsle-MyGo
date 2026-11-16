package market

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDetailsCapabilityProbingAndWebUIOnlyInstallGuard(t *testing.T) {
	service := NewServiceWithClient("", &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		status := 200
		switch {
		case r.URL.Host == "awesome-dsh-plugin.com":
			body = `{"updated":"2026-08-27","categories":{"ui":{"en":"UI"},"tools":{"en":"Tools"}},"plugins":[` +
				`{"owner":"fixture","name":"hybrid-ui","url":"https://github.com/fixture/hybrid-ui","npm":"hybrid-ui","category":"ui","description":{"en":"Hybrid plugin"}},` +
				`{"owner":"fixture","name":"pure-skill","url":"https://github.com/fixture/pure-skill","category":"tools","description":{"en":"Tool helper"}}` +
				`]}`
		case r.URL.Host == "registry.npmjs.org" && strings.Contains(r.URL.Path, "hybrid-ui"):
			body = `{"name":"hybrid-ui","version":"1.2.0","exports":{".":"./index.js","./client":"./client.js"},"dsh":{"bundle":{"patch":"cordis.patch.yml"},"client":"./client.js"}}`
		case r.URL.Host == "api.github.com" && strings.HasSuffix(r.URL.Path, "/readme"):
			body = "# Readme"
		case r.URL.Host == "api.github.com" && strings.Contains(r.URL.Path, "pure-skill/contents/package.json"):
			status = 404
		case r.URL.Host == "api.github.com" && strings.Contains(r.URL.Path, "pure-skill/contents/.mcp.json"):
			status = 404
		case r.URL.Host == "api.github.com" && strings.Contains(r.URL.Path, "pure-skill/contents/SKILL.md"):
			body = "---\nname: pure-skill\ndescription: Skill demo\n---\n"
		default:
			status = 404
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})})

	cat, err := service.List(context.Background(), true, "en")
	if err != nil || len(cat.Items) != 2 {
		t.Fatalf("list failed: %v %+v", err, cat.Items)
	}
	if cat.Items[0].CapabilityKind != CapabilityWebUIOnly || cat.Items[0].AllowsNativeMode() {
		t.Fatalf("expected category=ui hint to start web-ui-only: %+v", cat.Items[0])
	}

	detailsHybrid, err := service.Details(context.Background(), cat.Items[0], false)
	if err != nil || detailsHybrid.CapabilityKind != CapabilityHybrid || !detailsHybrid.NativeCompatible {
		t.Fatalf("expected npm manifest to upgrade hybrid-ui: %+v err=%v", detailsHybrid, err)
	}

	detailsSkill, err := service.Details(context.Background(), cat.Items[1], false)
	if err != nil || detailsSkill.CapabilityKind != CapabilitySkill || !detailsSkill.NativeCompatible {
		t.Fatalf("expected GitHub SKILL.md probe to detect skill: %+v err=%v", detailsSkill, err)
	}

	webOnly := &PackageCapabilities{Kind: CapabilityWebUIOnly, Guidance: "Web UI-only plugin requires the compatibility workbench."}
	res := InstallSpecContext(context.Background(), "github:fixture/ui-only", nil, InstallOptions{Capabilities: webOnly})
	if res.Error != webOnly.Guidance {
		t.Fatalf("expected explicit Web UI-only guard error, got: %+v", res)
	}
}
