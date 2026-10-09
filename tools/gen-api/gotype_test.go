package main

import (
	"strings"
	"testing"
)

// mapExpr parses one standalone zod expression and maps it to Go.
func mapExpr(t *testing.T, expr string, hint string) (string, string) {
	t.Helper()
	src := "import { z } from 'zod'\nconst S$schema = " + expr + "\nexport const TYPERT_REMOTE = { package: '@test/x', descriptors: [] }\n"
	pf := mustParse(t, src)
	g := newGoTypeer()
	g.schemas = pf.schemas
	g.curPkg = "x"
	g.inProg["S$schema"] = true // as rootType does when expanding the root schema
	exprGo, note := g.goType(pf.schemas["S$schema"], hint)
	return exprGo, note
}

func TestZodToGoMapping(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want string
	}{
		{"string", "z.string()", "string"},
		{"number", "z.number()", "float64"},
		{"boolean", "z.boolean()", "bool"},
		{"brandedString", "z.intersection(z.string(), z.unknown())", "string"},
		{"optionalString", "z.string().optional()", "string"},
		{"readonlyString", "z.string().readonly()", "string"},
		{"nullableString", "z.string().nullable()", "string"},
		{"literalUnion", "z.union([z.literal(\"positive\"), z.literal(\"negative\")])", "string"},
		{"nullOrString", "z.union([z.literal(null), z.intersection(z.string(), z.unknown())])", "string"},
		{"nullableUnion", "z.union([z.literal(null), z.intersection(z.string(), z.unknown())]).readonly()", "string"},
		{"boolUnion", "z.union([z.literal(true), z.literal(false)])", "bool"},
		{"arrayOfStrings", "z.array(z.string())", "[]string"},
		{"arrayOfNumbers", "z.array(z.number()).readonly()", "[]float64"},
		{"recordOfLazy", "z.record(z.string(), z.lazy(() => S$schema))", "map[string]any"},
		{"recordOfNumbers", "z.record(z.string(), z.number())", "map[string]float64"},
		{"jsonValue", "z.union([z.literal(null), z.string(), z.number(), z.literal(false), z.literal(true), z.array(z.lazy(() => S$schema)), z.record(z.string(), z.lazy(() => S$schema))])", "json.RawMessage"},
		{"unknown", "z.unknown()", "any"},
		{"tupleFallsBack", "z.tuple([z.string(), z.number()])", "json.RawMessage"},
		{"dateFallsBack", "z.date()", "json.RawMessage"},
		{"transformFallsBack", "z.string().transform((v) => v.length)", "json.RawMessage"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, _ := mapExpr(t, c.expr, "Thing")
			if got != c.want {
				t.Fatalf("goType(%s) = %q, want %q", c.expr, got, c.want)
			}
		})
	}
}

func TestObjectStructEmission(t *testing.T) {
	src := `import { z } from 'zod'
const S$schema = z.object({
  'id': z.intersection(z.string(), z.unknown()).readonly(),
  'revision': z.number().readonly(),
  'name': z.string().optional(),
  'tags': z.array(z.string()),
  'nested': z.object({
    'code': z.union([z.literal("a"), z.literal("b")]).readonly(),
  }).readonly().optional(),
})

export const TYPERT_REMOTE = {
  package: '@test/obj',
  descriptors: [],
}
`
	pf := mustParse(t, src)
	g := newGoTypeer()
	g.schemas = pf.schemas
	g.curPkg = "obj"
	g.docCtx = "test object"
	name := g.structType(pf.schemas["S$schema"], "Thing")
	if name != "Thing" {
		t.Fatalf("struct name = %q", name)
	}
	if len(g.defs) != 2 {
		t.Fatalf("defs = %d, want 2 (outer + nested)", len(g.defs))
	}
	var outer, nested string
	for _, d := range g.defs {
		switch {
		case strings.Contains(d.code, "type Thing struct {"):
			outer = d.code
		case strings.Contains(d.code, "type ThingNested struct {"):
			nested = d.code
		}
	}
	if outer == "" || nested == "" {
		var all strings.Builder
		for _, d := range g.defs {
			all.WriteString(d.code)
			all.WriteString("\n")
		}
		t.Fatalf("defs missing:\n%s", all.String())
	}
	for _, want := range []string{
		"Id string `json:\"id\"`",
		"Revision float64 `json:\"revision\"`",
		"Name *string `json:\"name,omitempty\"`",
		"Tags []string `json:\"tags\"`",
		"Nested *ThingNested `json:\"nested,omitempty\"`",
	} {
		if !strings.Contains(outer, want) {
			t.Fatalf("outer struct missing %q:\n%s", want, outer)
		}
	}
	if !strings.Contains(nested, "Code string `json:\"code\"`") {
		t.Fatalf("nested struct wrong:\n%s", nested)
	}
}

func TestNameFromSymbol(t *testing.T) {
	cases := []struct {
		symbol, method, arg string
		isResult            bool
		want                string
	}{
		{"@deepseek-ai/dsh-llm/types#LlmModelDiscoveryRequest", "discoverModels", "", false, "LlmModelDiscoveryRequest"},
		{"@deepseek-ai/dsh-llm#llm/discoverModels:settingsNs", "discoverModels", "settingsNs", false, "DiscoverModelsSettingsNs"},
		{"@deepseek-ai/dsh-llm#llm/discoverModels:result", "discoverModels", "", true, "DiscoverModelsResult"},
		{"@deepseek-ai/dsh-goal/client#GoalRef", "clear", "", false, "GoalRef"},
		{"@deepseek-ai/dsh-goal#goals/clear", "clear", "", false, "ClearArgs"},
		{"", "listProviders", "", true, "ListProvidersResult"},
	}
	for _, c := range cases {
		got := nameFromSymbol(c.symbol, c.method, c.arg, c.isResult)
		if got != c.want {
			t.Fatalf("nameFromSymbol(%q,%q,%q,%v) = %q, want %q", c.symbol, c.method, c.arg, c.isResult, got, c.want)
		}
	}
}
