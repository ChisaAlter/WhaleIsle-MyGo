package settings

import (
	"encoding/json"
	"testing"
)

var permissionSchema = json.RawMessage(`{
  "uid": 5,
  "refs": {
    "1": { "type": "const", "value": "read-only" },
    "2": { "type": "const", "value": "workspace-write" },
    "3": { "type": "const", "value": "danger-full-access" },
    "4": { "type": "union", "list": [1, 2, 3] },
    "5": { "type": "object", "dict": { "defaultPreset": 4 } }
  }
}`)

var labeledSchema = json.RawMessage(`{
  "uid": 3,
  "refs": {
    "1": { "type": "string", "meta": { "label": "API base", "description": "Override the default endpoint." } },
    "2": { "type": "object", "dict": { "endpoint": 1 } },
    "3": { "type": "object", "dict": { "endpoint": 1 } }
  }
}`)

func TestBuildFieldsMetaDescription(t *testing.T) {
	fields, err := BuildFields(labeledSchema)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 {
		t.Fatalf("fields = %d", len(fields))
	}
	f := fields[0]
	if f.Label != "API base" || f.Description != "Override the default endpoint." {
		t.Fatalf("field = %+v", f)
	}
}

func TestBuildFieldsEnumUnion(t *testing.T) {
	fields, err := BuildFields(permissionSchema)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 1 {
		t.Fatalf("fields = %d, want 1", len(fields))
	}
	f := fields[0]
	if f.Key != "defaultPreset" || f.Kind != FieldEnum {
		t.Fatalf("field = %+v", f)
	}
	if len(f.EnumOptions) != 3 {
		t.Fatalf("enum opts = %v", f.EnumOptions)
	}
}

func TestPatchValueBoolAndEnum(t *testing.T) {
	raw, err := PatchValue(FieldEnum, "read-only")
	if err != nil || string(raw) != `"read-only"` {
		t.Fatalf("enum patch: %q %v", raw, err)
	}
	raw, err = PatchValue(FieldBool, "true")
	if err != nil || string(raw) != "true" {
		t.Fatalf("bool patch: %q %v", raw, err)
	}
}

func TestValueDraft(t *testing.T) {
	value := json.RawMessage(`{"defaultPreset":"workspace-write","count":3}`)
	if got := ValueDraft(value, "defaultPreset", FieldEnum); got != "workspace-write" {
		t.Fatalf("draft = %q", got)
	}
	if got := ValueDraft(value, "count", FieldNumber); got != "3" {
		t.Fatalf("number draft = %q", got)
	}
}
