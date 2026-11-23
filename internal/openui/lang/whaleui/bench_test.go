package whaleui

import (
	"strconv"
	"strings"
	"testing"

	"whaleisle/internal/openui/lang"
)

// benchProgram builds a realistic Whale UI program of about size bytes:
// root first, state, sections with tables, charts, inputs and @Each lists.
func benchProgram(size int) string {
	var b strings.Builder
	var sections []string
	body := strings.Builder{}
	body.WriteString("$filter = \"\"\n$qty = 2\n$items = [{id:1,name:\"Alpha\",price:12.5,done:false},{id:2,name:\"Beta\",price:8,done:true},{id:3,name:\"Gamma\",price:30,done:false}]\n")
	for i := 0; body.Len() < size-200; i++ {
		n := strconv.Itoa(i)
		sections = append(sections, "s"+n)
		body.WriteString("s" + n + " = Card([Title(\"Section " + n + "\",\"sm\"), Row([Text(\"Total \" + @Format(t" + n + ",\"currency\",2,\"USD\")), Spacer(), Badge(@Count(open" + n + ") + \" open\",\"info\")],2), list" + n + ", Table([\"Name\",\"Price\"], rows" + n + "), Details(\"More\",[Input($qty,\"Quantity\",\"number\"), Slider($qty,0,10,1,\"Qty\")])],\"outline\")\n")
		body.WriteString("open" + n + " = @Filter($items, \"done\", \"==\", false)\n")
		body.WriteString("t" + n + " = @Sum($items.price) * @Clamp(@Default($qty,0),0,100)\n")
		body.WriteString("list" + n + " = Col(@Each(@Sort($items,\"price\",\"desc\"), \"it\", Row([Checkbox($done" + n + ", it.name), Text(@Upper(it.name) + \" \" + @Round(it.price * $qty, 2)), Button(\"Pick\", @Set($filter, it.name), \"ghost\")])))\n")
		body.WriteString("rows" + n + " = @Each($items, \"r\", [r.name, @Format(r.price, \"number\", 2)])\n")
	}
	b.WriteString("root = Col([Title(\"Benchmark\",\"md\"), " + strings.Join(sections, ", ") + "])\n")
	b.WriteString(body.String())
	return b.String()
}

func TestBenchProgramRenders(t *testing.T) {
	prog := benchProgram(20 << 10)
	_, f := render(t, prog)
	if f.Root == nil || len(errorsOf(f)) > 0 {
		t.Fatalf("bench program: %v", errorsOf(f))
	}
	if len(prog) < 19<<10 || len(prog) > 21<<10 {
		t.Fatalf("size %d", len(prog))
	}
}

// BenchmarkParseEval20KB measures a batch parse plus full evaluation of a
// 20 KB program.
func BenchmarkParseEval20KB(b *testing.B) {
	lib := mustLib(b)
	prog := benchProgram(20 << 10)
	b.SetBytes(int64(len(prog)))
	b.ReportAllocs()
	for b.Loop() {
		in := lang.NewInstance(lib, lang.Options{})
		if f := in.Update(lang.Update{Program: prog}); f.Root == nil {
			b.Fatal("no root")
		}
	}
}

// BenchmarkStream200 measures a streaming sequence of 200 growing revisions
// of the 20 KB program (parse + evaluation per revision), ending the stream.
func BenchmarkStream200(b *testing.B) {
	lib := mustLib(b)
	prog := benchProgram(20 << 10)
	const steps = 200
	cuts := make([]int, steps)
	for i := range cuts {
		cuts[i] = len(prog) * (i + 1) / steps
	}
	b.ReportAllocs()
	for b.Loop() {
		in := lang.NewInstance(lib, lang.Options{})
		var f *lang.Frame
		for i, c := range cuts {
			f = in.Update(lang.Update{Revision: uint64(i + 1), Program: prog[:c], Streaming: i < steps-1})
		}
		if f.Root == nil {
			b.Fatal("no root")
		}
	}
}

// BenchmarkRender20KB measures re-evaluation after a state change.
func BenchmarkRender20KB(b *testing.B) {
	lib := mustLib(b)
	in := lang.NewInstance(lib, lang.Options{})
	in.Update(lang.Update{Program: benchProgram(20 << 10)})
	b.ReportAllocs()
	i := 0.0
	for b.Loop() {
		i++
		_ = in.SetState("$qty", i)
		if in.Render().Root == nil {
			b.Fatal("no root")
		}
	}
}
