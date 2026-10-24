package chat

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

func TestToolBodyFormatPresenceAndCodeSelection(t *testing.T) {
	for _, tc := range []struct {
		name, arguments, want string
		present               bool
	}{
		{"empty raw", "", "", false},
		{"whitespace raw", " \t\r\n", " \t\r\n", true},
		{"incomplete object", ` {"code":`, ` {"code":`, true},
		{"trailing data", `{"code":"ignored"} true`, `{"code":"ignored"} true`, true},
		{"invalid trailing comma", `{"code":"ignored",}`, `{"code":"ignored",}`, true},
		{"non JSON number", `NaN`, `NaN`, true},
		{"non JSON whitespace", "\u00a0null", "\u00a0null", true},
		{"exact authored code", `{"code":"  const text = '<>&';\n\n","description":"run"}`, "  const text = '<>&';\n\n", true},
		{"whitespace code", `{"code":" \t\r\n"}`, " \t\r\n", true},
		{"code containing null", `{"code":"\u0000"}`, "\x00", true},
		{"decoded key and code", `{"co\u0064e":"\u0061\ud83d\ude00"}`, "a😀", true},
		{"last code value", `{"code":"first","code":"last\n"}`, "last\n", true},
		{"last code empty", `{"code":"first","code":""}`, "{\n  \"code\": \"\"\n}", true},
		{"last code nonstring", `{"code":"first","code":false}`, "{\n  \"code\": false\n}", true},
		{"empty code", `{"code":"","description":"run"}`, "{\n  \"code\": \"\",\n  \"description\": \"run\"\n}", true},
		{"numeric code", `{"code":42}`, "{\n  \"code\": 42\n}", true},
		{"null code", `{"code":null}`, "{\n  \"code\": null\n}", true},
		{"string root", ` "  authored\n" `, `"  authored\n"`, true},
		{"null root", " \nnull\t", "null", true},
		{"true root", `true`, "true", true},
		{"false root", `false`, "false", true},
		{"empty object", ` { } `, "{}", true},
		{"empty array", ` [ ] `, "[]", true},
		{"array has no code property", `[{"code":"keep envelope"},true,null]`, "[\n  {\n    \"code\": \"keep envelope\"\n  },\n  true,\n  null\n]", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, present := formatToolCodeProgram(tc.arguments)
			if got != tc.want || present != tc.present {
				t.Fatalf("formatToolCodeProgram(%q) = %q, %v; want %q, %v", tc.arguments, got, present, tc.want, tc.present)
			}
		})
	}
}

func TestToolBodyFormatObjectOrderingAndDuplicateValues(t *testing.T) {
	arguments := `{"z":"first","4294967295":"ordinary first","10":"ten","2":"two","01":"leading zero","0":"zero","4294967294":"largest index","-0":"negative zero","1.0":"decimal","a":{"tail":1,"3":"three","1":"one","tail":2},"z":"last","\u0032":"two final"}`
	want := `{
  "0": "zero",
  "2": "two final",
  "10": "ten",
  "4294967294": "largest index",
  "z": "last",
  "4294967295": "ordinary first",
  "01": "leading zero",
  "-0": "negative zero",
  "1.0": "decimal",
  "a": {
    "1": "one",
    "3": "three",
    "tail": 2
  }
}`
	if got, present := formatToolCodeProgram(arguments); !present || got != want {
		t.Fatalf("ordered body = %q, %v; want %q, true", got, present, want)
	}
	// JSON.parse treats __proto__ as an ordinary own property, including when
	// its value happens to contain a code property.
	arguments = `{"__proto__":{"code":"not inherited"},"constructor":null}`
	want = "{\n  \"__proto__\": {\n    \"code\": \"not inherited\"\n  },\n  \"constructor\": null\n}"
	if got, _ := formatToolCodeProgram(arguments); got != want {
		t.Fatalf("special property body = %q; want %q", got, want)
	}
}

func TestToolBodyFormatStringifyStrings(t *testing.T) {
	for _, tc := range []struct {
		name, arguments, want string
	}{
		{"HTML and decoded Unicode", `{"text":"\u003c>&\u0061\ud83d\ude00"}`, "{\n  \"text\": \"<>&a😀\"\n}"},
		{"literal line separators", `"\u2028\u2029"`, "\"\u2028\u2029\""},
		{"literal backslash escape", `"literal \\u2028; actual \u2028"`, `"literal \\u2028; actual ` + "\u2028\""},
		{"JSON controls", `"\u0000\u0001\b\f\n\r\t\u001f"`, `"\u0000\u0001\b\f\n\r\t\u001f"`},
		{"slash quote and backslash", `"\/\"\\"`, `"/\"\\"`},
		{"non ASCII property name", `{"\u003c\u2028":"\u4e16\u754c"}`, "{\n  \"<\u2028\": \"世界\"\n}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, present := formatToolCodeProgram(tc.arguments); !present || got != tc.want {
				t.Fatalf("string body = %q, %v; want %q, true", got, present, tc.want)
			}
		})
	}
}

func TestToolBodyFormatStringifyNumbers(t *testing.T) {
	for _, tc := range []struct{ arguments, want string }{
		{"-0", "0"},
		{"-0.0e+5", "0"},
		{"1.00", "1"},
		{"1.0000000000000001", "1"},
		{"9007199254740993", "9007199254740992"},
		{"-9007199254740993", "-9007199254740992"},
		{"1000000000000000128", "1000000000000000100"},
		{"333333333.33333329", "333333333.3333333"},
		{"1e20", "100000000000000000000"},
		{"1e21", "1e+21"},
		{"-1e21", "-1e+21"},
		{"1e-6", "0.000001"},
		{"1e-7", "1e-7"},
		{"-1e-7", "-1e-7"},
		{"1.23e-7", "1.23e-7"},
		{"5e-324", "5e-324"},
		{"1e-400", "0"},
		{"-1e-400", "0"},
		{"1.7976931348623157e308", "1.7976931348623157e+308"},
		{"1e400", "null"},
		{"-1e400", "null"},
	} {
		t.Run(tc.arguments, func(t *testing.T) {
			if got, present := formatToolCodeProgram(tc.arguments); !present || got != tc.want {
				t.Fatalf("numeric body = %q, %v; want %q, true", got, present, tc.want)
			}
		})
	}
	arguments := `{"numbers":[-0,9007199254740993,1e400,1e-7],"code":1e20}`
	want := "{\n  \"numbers\": [\n    0,\n    9007199254740992,\n    null,\n    1e-7\n  ],\n  \"code\": 100000000000000000000\n}"
	if got, _ := formatToolCodeProgram(arguments); got != want {
		t.Fatalf("nested numbers = %q; want %q", got, want)
	}
}

func TestToolBodyFormatRetainsRawRecord(t *testing.T) {
	arguments := " {\"z\":9007199254740993,\"code\":\"  exact();\\n\",\"z\":-0} \r\n"
	message := session.Message{Raw: json.RawMessage(arguments)}
	original := bytes.Clone(message.Raw)
	for range 2 {
		if got, present := formatToolCodeProgram(string(message.Raw)); !present || got != "  exact();\n" {
			t.Fatalf("program = %q, %v", got, present)
		}
		if !bytes.Equal(message.Raw, original) {
			t.Fatal("formatting rewrote the retained raw record")
		}
	}
	message.Raw = json.RawMessage(` {"z":9007199254740993,"a":"\u003c>&","z":-0} `)
	original = bytes.Clone(message.Raw)
	want := "{\n  \"z\": 0,\n  \"a\": \"<>&\"\n}"
	if got, _ := formatToolCodeProgram(string(message.Raw)); got != want {
		t.Fatalf("formatted body = %q; want %q", got, want)
	}
	if !bytes.Equal(message.Raw, original) {
		t.Fatal("JSON formatting rewrote the retained raw record")
	}
}

func TestToolBodyFormatNativeJSONLimits(t *testing.T) {
	// Go cannot represent JavaScript's lone UTF-16 code unit as valid UTF-8.
	// Keep this limitation explicit rather than claiming byte-for-byte parity.
	if got, _ := formatToolCodeProgram(`"\ud800"`); got != "\"\ufffd\"" {
		t.Fatalf("native lone surrogate = %q; want replacement character", got)
	}
	if got, _ := formatToolCodeProgram(`{"code":"\ud800"}`); got != "\ufffd" {
		t.Fatalf("native code lone surrogate = %q; want replacement character", got)
	}
	deep := strings.Repeat("[", 10001) + "0" + strings.Repeat("]", 10001)
	if got, present := formatToolCodeProgram(deep); !present || got != deep {
		t.Fatal("input beyond encoding/json's nesting limit did not retain raw text")
	}
}
