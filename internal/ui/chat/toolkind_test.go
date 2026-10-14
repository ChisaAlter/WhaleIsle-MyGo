package chat

import "testing"

func TestClassifyToolName(t *testing.T) {
	cases := []struct {
		name string
		kind ToolKind
		chip string
	}{
		{"Bash", ToolKindBash, "bash"},
		{"read_file", ToolKindRead, "read"},
		{"WebFetch", ToolKindWeb, "web"},
		{"todo_write", ToolKindTodo, "todo"},
		{"Task", ToolKindSubagent, "agent"},
		{"custom", ToolKindGeneric, "tool"},
	}
	for _, tc := range cases {
		kind, chip := ClassifyToolName(tc.name)
		if kind != tc.kind || chip != tc.chip {
			t.Fatalf("%s -> %s/%s, want %s/%s", tc.name, kind, chip, tc.kind, tc.chip)
		}
	}
}
