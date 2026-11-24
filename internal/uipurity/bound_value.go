package uipurity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Bound-value guard: mygo's deferred-writeback controls (Checkbox, TextInput,
// Slider, Select, Split, ...) write the bound pointer inside applyInputs(),
// which runs when an element query (Changed, Clicked, Submitted, Shortcut,
// ...) executes or at end of frame. Reading the bound value between the
// control call and the next query sees last frame's value; binding a stack
// local nobody queries drops the user's input entirely. This rule flags:
//
//	a) a stack local passed as &local to a deferred control whose value is
//	   read before the next flushing query — or never read at all;
//	b) a before-snapshot (`before := expr`) compared against expr after a
//	   deferred control bound &expr without an intervening query;
//	c) a pointer passed directly (ui.TextArea(c, p)) whose pointee is
//	   dereferenced (*p) before the next flushing query.
//
// flushCall is every element query that runs applyInputs() first (verified
// against third_party/mygo/ui: input.go Changed/Submitted, api_gen.go
// Clicked/Clicks/DoubleClicked/RightClicked/Shortcut/OverlayShortcut,
// page_shortcut.go, physical_shortcut.go, combobox.go Chosen). Focused,
// Composing and Hovered deliberately do NOT flush.
var flushCall = map[string]bool{
	"Changed":          true,
	"Submitted":        true,
	"Clicked":          true,
	"Clicks":           true,
	"DoubleClicked":    true,
	"RightClicked":     true,
	"Shortcut":         true,
	"OverlayShortcut":  true,
	"PageShortcut":     true,
	"PhysicalShortcut": true,
	"Chosen":           true,
	// Project helpers that return an inner control's Changed() and so
	// flush pending inputs as a side effect.
	"wallpaperSlider":       true,
	"wallpaperSliderStep":   true,
	"wallpaperChoiceSelect": true,
	"appearanceColorWell":   true,
}

// deferredCallback prefixes mark callback registrars whose FuncLit argument
// runs during or after input processing — applyInputs has already written
// every pending binding by then, so bound values read inside are fresh. The
// codebase uses .OnXxx methods and OnXxx:/HandlerXxx option fields broadly.
func deferredCallback(name string) bool {
	return strings.HasPrefix(name, "On") || strings.HasPrefix(name, "Handle")
}

// deferredControl is every constructor that registers a value binding written
// in a deferred afterInput/onValueInput callback. Names are matched by
// selector, covering ui.* (incl. Base/Buffer variants) and the coss.* wrappers
// that delegate to them.
var deferredControl = map[string]bool{
	// ui.* toggles and choices
	"Checkbox": true, "CheckboxBase": true,
	"Switch": true, "SwitchBase": true,
	"Toggle": true, "ToggleBase": true,
	"Radio": true, "RadioBase": true,
	"Select": true, "SelectBase": true,
	"SegmentedBase": true,
	"Tabs":          true, "TabsBase": true,
	"Collapsible": true, "CollapsibleBase": true,
	"Combobox": true, "ComboboxBase": true,
	// ui.* text
	"TextInput": true, "TextInputBase": true,
	"TextInputBuffer": true, "TextInputBufferBase": true,
	"TextArea": true, "TextAreaBase": true,
	"TextAreaBuffer": true, "TextAreaBufferBase": true,
	// ui.* numbers, time, color
	"Slider": true, "SliderBase": true, "StepSlider": true,
	"NumberInput": true, "TimeInput": true, "DatePicker": true,
	"ColorPicker": true,
	// ui.* splits bind a size/fraction written by the divider
	"Split": true, "SplitVertical": true,
	"SplitFraction": true, "SplitVerticalFraction": true,
	// coss.* wrappers delegating to the *Base controls above
	"SegmentedControl": true, "Input": true, "InputGroup": true,
}

// BoundViolation is one flagged use of a deferred control's bound value.
type BoundViolation struct {
	File string
	Line int
	Msg  string
}

func (v BoundViolation) String() string {
	return v.File + ":" + strconv.Itoa(v.Line) + ": " + v.Msg
}

// ScanBoundValue walks the given directories (skipping _test.go files) and
// returns every bound-value violation, sorted by file and line.
func ScanBoundValue(dirs ...string) ([]BoundViolation, error) {
	var files []string
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	var out []BoundViolation
	for _, path := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, err
		}
		rel := filepath.ToSlash(path)
		for _, v := range scanBoundAST(fset, f, rel) {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

// scanBoundSource is the in-memory form for unit tests.
func scanBoundSource(name, src string) ([]BoundViolation, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil, err
	}
	return scanBoundAST(fset, f, name), nil
}

// exprKey renders an identifier or selector chain ("x", "s.Field") used to
// recognize the same bound expression later. Anything else returns "".
func exprKey(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if base := exprKey(e.X); base != "" {
			return base + "." + e.Sel.Name
		}
	}
	return ""
}

// binding records one deferred writeback onto the named expression.
type binding struct {
	key     string // exprKey of the bound value ("x", "s.Title", "m.Draft")
	local   bool   // bound via &x where x is a stack local (pattern a)
	pos     token.Pos
	flushed bool
	read    bool // value observed after the binding
	// reported guards against double-flagging one binding.
	reported bool
}

// bindingSet keeps local (&x) bindings and pointer/field bindings in separate
// lists so a bare pointer argument never shadows an &x binding for the same
// identifier.
type bindingSet struct {
	local []*binding
	ptr   []*binding
}

func (bs *bindingSet) all() []*binding { return append(append([]*binding{}, bs.local...), bs.ptr...) }

// funcScan walks one top-level function in source order. Nested FuncLit
// bodies share the stream: mygo runs Children/style closures during the
// enclosing call, so source order is execution order for the build pass.
type funcScan struct {
	fset       *token.FileSet
	file       string
	locals     map[string]bool        // declared stack variables
	bound      map[string]*bindingSet // live bindings per bound expression key
	snaps      map[string]string      // var name -> exprKey it was assigned
	readOnly   map[token.Pos]bool     // &x arg positions under a ReadOnly(true) chain
	exempt     map[string]int         // bound locals readable in this context
	exemptAll  int                    // inside a built-element chain's config arg
	elemVars   map[string][]string    // vars holding a bound element -> bound local names
	deferArg   bool                   // current arg runs after input processing
	deferDepth int                    // inside a deferred-callback closure
	flagReads  int                    // reads in decision positions (conds, assigns)
	out        *[]BoundViolation
}

func (s *funcScan) fail(v BoundViolation) { *s.out = append(*s.out, v) }

func (s *funcScan) bind(key string, local bool, pos token.Pos) {
	b := &binding{key: key, local: local, pos: pos}
	bs := s.bound[key]
	if bs == nil {
		bs = &bindingSet{}
		s.bound[key] = bs
	}
	if local {
		bs.local = append(bs.local, b)
	} else {
		bs.ptr = append(bs.ptr, b)
	}
}

// liveLocal returns the newest &x stack-local binding for key.
func (s *funcScan) liveLocal(key string) *binding {
	if bs := s.bound[key]; bs != nil && len(bs.local) > 0 {
		return bs.local[len(bs.local)-1]
	}
	return nil
}

// livePtr returns the newest pointer/field binding for key.
func (s *funcScan) livePtr(key string) *binding {
	if bs := s.bound[key]; bs != nil && len(bs.ptr) > 0 {
		return bs.ptr[len(bs.ptr)-1]
	}
	return nil
}

func (s *funcScan) flush() {
	for _, bs := range s.bound {
		for _, b := range bs.all() {
			b.flushed = true
		}
	}
}

// unbind drops bindings for a variable that was redeclared or assigned over:
// the pending deferred write now targets a dead value.
func (s *funcScan) unbind(key string) { delete(s.bound, key) }

// flagRead reports a pre-flush read of the given binding.
func (s *funcScan) flagRead(b *binding, pos token.Pos) {
	b.read = true
	if b.flushed || b.reported {
		return
	}
	b.reported = true
	s.fail(BoundViolation{
		File: s.file,
		Line: s.fset.Position(pos).Line,
		Msg: "bound value " + strconv.Quote(b.key) + " read before an input query flushed it: " +
			"reads see last frame's value — call Changed()/Submitted()/Clicked() first",
	})
}

// markRead records a use that consumes the bound value without being a stale
// read itself (&x aliases, writes): keeps the never-read check accurate.
func (s *funcScan) markRead(key string) {
	if bs := s.bound[key]; bs != nil {
		for _, b := range bs.local {
			b.read = true
		}
	}
}

// readLocal observes one use of a bound stack local: it marks every live
// binding of the variable read (one read consumes all pending writes) and
// flags the newest binding when the read happens in a decision position —
// a condition, an assignment, ++/--, a channel send — before any input query
// has flushed pending writes. Configuration reads (call arguments building
// labels, tooltips, Disabled flags, the control's own chain) are exempt:
// showing last frame's value for one frame is the immediate-mode norm.
func (s *funcScan) readLocal(key string, pos token.Pos) {
	bs := s.bound[key]
	if bs == nil || len(bs.local) == 0 {
		return
	}
	for _, b := range bs.local {
		b.read = true
	}
	if s.flagReads == 0 || s.deferDepth > 0 || s.exempt[key] > 0 || s.exemptAll > 0 {
		return
	}
	s.flagRead(bs.local[len(bs.local)-1], pos)
}

// finish flags bound stack locals nobody ever read or queried: the user's
// input is deferred-written into a dead variable and silently dropped. A
// flushed binding (the element's Changed()/Clicked() was used as the trigger)
// is exempt — the query consumed the event even if the value is unused.
func (s *funcScan) finish() {
	for _, bs := range s.bound {
		for _, b := range bs.local {
			if !b.read && !b.flushed && !b.reported {
				s.fail(BoundViolation{
					File: s.file,
					Line: s.fset.Position(b.pos).Line,
					Msg: "stack local " + strconv.Quote(b.key) + " bound to a deferred input but never read: " +
						"user input is written to a dead variable — bind persistent state or use the element's Changed()",
				})
			}
		}
	}
}

func scanBoundAST(fset *token.FileSet, f *ast.File, file string) []BoundViolation {
	var out []BoundViolation
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		s := &funcScan{
			fset:     fset,
			file:     file,
			locals:   map[string]bool{},
			bound:    map[string]*bindingSet{},
			snaps:    map[string]string{},
			readOnly: map[token.Pos]bool{},
			exempt:   map[string]int{},
			elemVars: map[string][]string{},
			out:      &out,
		}
		// Parameters are stack locals too: &param writes the caller-passed
		// copy and drops the input.
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				for _, name := range field.Names {
					s.locals[name.Name] = true
				}
			}
		}
		s.markReadOnlyArgs(fn.Body)
		s.walkBlock(fn.Body)
		s.finish()
	}
	return out
}

func (s *funcScan) walkBlock(b *ast.BlockStmt) {
	for _, stmt := range b.List {
		s.walkStmt(stmt)
	}
}

func (s *funcScan) walkStmt(n ast.Stmt) {
	switch n := n.(type) {
	case *ast.AssignStmt:
		// Evaluate RHS before recording LHS bindings, matching Go order.
		// Assignment reads are decision-position reads: `n.Ratio = size` or
		// `before := expr` persist last frame's value. A FuncLit assigned to
		// a variable is a stored callback — it runs after input processing.
		for _, e := range n.Rhs {
			saved := s.deferArg
			if _, ok := e.(*ast.FuncLit); ok {
				s.deferArg = true
			}
			s.flagReads++
			s.walkExpr(e)
			s.flagReads--
			s.deferArg = saved
		}
		for i, lhs := range n.Lhs {
			if id, ok := lhs.(*ast.Ident); ok {
				if n.Tok == token.DEFINE {
					s.locals[id.Name] = true
					s.unbind(id.Name)
				}
				if len(n.Rhs) == len(n.Lhs) {
					// Track before-snapshots: name := expr remembers exprKey.
					if key := exprKey(n.Rhs[i]); key != "" {
						s.snaps[id.Name] = key
					} else {
						delete(s.snaps, id.Name)
					}
					// v := <control call> remembers which locals v's
					// element binds, for receiver-chain exemptions.
					for _, nm := range s.spineLocalBinds(n.Rhs[i]) {
						if s.elemVars[id.Name] == nil {
							s.elemVars[id.Name] = []string{}
						}
						s.elemVars[id.Name] = append(s.elemVars[id.Name], nm)
					}
				}
				// x = v overwrites a pending bound value: drop the binding.
				if n.Tok == token.ASSIGN {
					s.unbind(id.Name)
				}
				continue
			}
			// s.F = v / s[i] = v: writes, not stale reads — walk for calls
			// without flagging the written-into base.
			s.walkExpr(lhs)
		}
	case *ast.DeclStmt:
		if d, ok := n.Decl.(*ast.GenDecl); ok {
			for _, spec := range d.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for _, e := range vs.Values {
						saved := s.deferArg
						if _, ok := e.(*ast.FuncLit); ok {
							s.deferArg = true
						}
						s.flagReads++
						s.walkExpr(e)
						s.flagReads--
						s.deferArg = saved
					}
					for _, name := range vs.Names {
						s.locals[name.Name] = true
						s.unbind(name.Name)
						delete(s.snaps, name.Name)
					}
				}
			}
		}
	case *ast.ExprStmt:
		s.walkExpr(n.X)
	case *ast.ReturnStmt:
		// Returning a bound local hands the pre-flush value to the caller
		// (the historyTurnPicker `return selected, trigger.Changed() &&
		// selected >= 0` idiom): not a decision position, observe only.
		for _, e := range n.Results {
			s.walkExpr(e)
		}
	case *ast.IfStmt:
		if n.Init != nil {
			s.walkStmt(n.Init)
		}
		s.flagReads++
		s.walkExpr(n.Cond)
		s.flagReads--
		s.walkBlock(n.Body)
		if n.Else != nil {
			switch els := n.Else.(type) {
			case *ast.BlockStmt:
				s.walkBlock(els)
			case *ast.IfStmt:
				s.walkStmt(els)
			}
		}
	case *ast.SwitchStmt:
		if n.Init != nil {
			s.walkStmt(n.Init)
		}
		if n.Tag != nil {
			s.flagReads++
			s.walkExpr(n.Tag)
			s.flagReads--
		}
		// Case bodies are alternatives: a binding created inside one does
		// not exist in its siblings.
		saved := s.bound
		for _, stmt := range n.Body.List {
			s.bound = copyBound(saved)
			cc := stmt.(*ast.CaseClause)
			s.flagReads++
			for _, e := range cc.List {
				s.walkExpr(e)
			}
			s.flagReads--
			for _, st := range cc.Body {
				s.walkStmt(st)
			}
		}
		s.bound = saved
	case *ast.TypeSwitchStmt:
		if n.Init != nil {
			s.walkStmt(n.Init)
		}
		s.walkStmt(n.Assign)
		saved := s.bound
		for _, stmt := range n.Body.List {
			s.bound = copyBound(saved)
			cc := stmt.(*ast.CaseClause)
			for _, st := range cc.Body {
				s.walkStmt(st)
			}
		}
		s.bound = saved
	case *ast.SelectStmt:
		for _, stmt := range n.Body.List {
			cc := stmt.(*ast.CommClause)
			if cc.Comm != nil {
				s.walkStmt(cc.Comm)
			}
			for _, st := range cc.Body {
				s.walkStmt(st)
			}
		}
	case *ast.ForStmt:
		if n.Init != nil {
			s.walkStmt(n.Init)
		}
		if n.Cond != nil {
			s.flagReads++
			s.walkExpr(n.Cond)
			s.flagReads--
		}
		s.walkBlock(n.Body)
		if n.Post != nil {
			s.walkStmt(n.Post)
		}
	case *ast.RangeStmt:
		s.flagReads++
		s.walkExpr(n.X)
		s.flagReads--
		for _, e := range []ast.Expr{n.Key, n.Value} {
			if id, ok := e.(*ast.Ident); ok && n.Tok == token.DEFINE {
				s.locals[id.Name] = true
			}
		}
		s.walkBlock(n.Body)
	case *ast.GoStmt, *ast.DeferStmt:
		var call *ast.CallExpr
		if g, ok := n.(*ast.GoStmt); ok {
			call = g.Call
		} else {
			call = n.(*ast.DeferStmt).Call
		}
		if lit, ok := call.Fun.(*ast.FuncLit); ok {
			s.deferDepth++
			s.walkFuncLit(lit)
			s.deferDepth--
		} else {
			s.walkExpr(call)
		}
	case *ast.IncDecStmt:
		s.flagReads++
		s.walkExpr(n.X)
		s.flagReads--
	case *ast.SendStmt:
		s.flagReads++
		s.walkExpr(n.Chan)
		s.walkExpr(n.Value)
		s.flagReads--
	case *ast.LabeledStmt:
		s.walkStmt(n.Stmt)
	case *ast.BlockStmt:
		s.walkBlock(n)
	}
}

// walkExpr processes one expression in source order: calls record bindings or
// flushes before their arguments are walked, mirroring evaluation order.
func (s *funcScan) walkExpr(n ast.Expr) {
	switch n := n.(type) {
	case *ast.CallExpr:
		// Go evaluates the receiver/function expression first, then the
		// arguments, then the call — so `ui.Checkbox(c,&v).Changed()` binds,
		// then flushes, exactly in this walk order.
		var spine []string
		if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
			s.walkExpr(sel.X)
			// Locals bound by the control at the root of this method chain
			// may be read inside chain arguments (`.Width(len(v)*8)`) —
			// the chain still runs during construction. The same holds for
			// methods called on a variable holding a bound element
			// (input.SetTextSelection(0, len(draft)), tabs.List.Children).
			spine = s.spineLocalBinds(sel.X)
			ast.Inspect(sel.X, func(x ast.Node) bool {
				if id, ok := x.(*ast.Ident); ok {
					spine = append(spine, s.elemVars[id.Name]...)
				}
				return true
			})
		} else {
			s.walkExpr(n.Fun)
		}
		name := callName(n)
		for _, nm := range spine {
			s.exempt[nm]++
		}
		// Method calls on a built element chain (ui.Button(...).Disabled(x))
		// take configuration: bound locals inside their arguments configure
		// the element for this pass, so a one-frame-old value is the norm.
		// FuncLit args are exempt only via the callee name (On*/Handle*).
		_, chainRecv := n.Fun.(*ast.SelectorExpr)
		chainConfig := false
		if chainRecv {
			_, chainConfig = n.Fun.(*ast.SelectorExpr).X.(*ast.CallExpr)
		}
		for _, arg := range n.Args {
			saved := s.deferArg
			s.deferArg = deferredCallback(name)
			_, isLit := arg.(*ast.FuncLit)
			if chainConfig && !isLit {
				s.exemptAll++
			}
			s.walkExpr(arg)
			if chainConfig && !isLit {
				s.exemptAll--
			}
			s.deferArg = saved
		}
		for _, nm := range spine {
			s.exempt[nm]--
		}
		if deferredControl[name] {
			s.bindArgs(n)
		} else if flushCall[name] {
			s.flush()
		}
	case *ast.BinaryExpr:
		if n.Op == token.EQL || n.Op == token.NEQ {
			s.checkSnapshotCompare(n)
		}
		s.walkExpr(n.X)
		s.walkExpr(n.Y)
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			// &x aliases the bound value rather than reading it — taking
			// the address still observes the variable (Patch{F: &x}), so
			// the deferred write is not dropped; it is never a stale read.
			if id, ok := n.X.(*ast.Ident); ok {
				s.markRead(id.Name)
			}
			if sel, ok := n.X.(*ast.SelectorExpr); ok {
				s.walkExpr(sel.X)
			}
			return
		}
		s.walkExpr(n.X)
	case *ast.Ident:
		if s.locals[n.Name] {
			s.readLocal(n.Name, n.Pos())
		}
	case *ast.SelectorExpr:
		s.walkExpr(n.X)
	case *ast.IndexExpr:
		s.walkExpr(n.X)
		s.walkExpr(n.Index)
	case *ast.IndexListExpr:
		s.walkExpr(n.X)
		for _, e := range n.Indices {
			s.walkExpr(e)
		}
	case *ast.SliceExpr:
		s.walkExpr(n.X)
		for _, e := range []ast.Expr{n.Low, n.High, n.Max} {
			if e != nil {
				s.walkExpr(e)
			}
		}
	case *ast.StarExpr:
		// *p dereferences a bound value: flag the stale read.
		if key := exprKey(n.X); key != "" {
			if b := s.livePtr(key); b != nil {
				s.flagRead(b, n.Pos())
			} else if b := s.liveLocal(key); b != nil {
				s.readLocal(key, n.Pos())
			}
		}
		s.walkExpr(n.X)
	case *ast.ParenExpr:
		s.walkExpr(n.X)
	case *ast.TypeAssertExpr:
		s.walkExpr(n.X)
	case *ast.FuncLit:
		// An inline FuncLit arg runs when its callee says so: deferred for
		// .On*/.Handle* registrars (fresh reads), inline for builders like
		// Children and Split panes (same staleness as linear code).
		deferred := s.deferArg || s.deferDepth > 0
		if deferred {
			s.deferDepth++
		}
		s.walkFuncLit(n)
		if deferred {
			s.deferDepth--
		}
	case *ast.CompositeLit:
		if n.Type != nil {
			s.walkExpr(n.Type)
		}
		for _, e := range n.Elts {
			s.walkExpr(e)
		}
	case *ast.KeyValueExpr:
		saved := s.deferArg
		if id, ok := n.Key.(*ast.Ident); ok && deferredCallback(id.Name) {
			s.deferArg = true
		}
		s.walkExpr(n.Value)
		s.deferArg = saved
	}
}

// walkFuncLit walks a closure body sharing the caller's binding stream but
// scoping its own declarations.
func (s *funcScan) walkFuncLit(n *ast.FuncLit) {
	saved := s.locals
	s.locals = copyLocals(saved)
	if n.Type.Params != nil {
		for _, field := range n.Type.Params.List {
			for _, name := range field.Names {
				s.locals[name.Name] = true
			}
		}
	}
	s.walkBlock(n.Body)
	s.locals = saved
}

// bindArgs records deferred writebacks for a control call's &x arguments:
// &x on a stack local is pattern (a); &s.F is a persistent field binding
// checked only by snapshot compares. Bare pointer arguments (p, m.Draft) are
// not bound: reading *p to render the current value is the immediate-mode
// norm, and stale decision reads are covered by the compare rule.
func (s *funcScan) bindArgs(call *ast.CallExpr) {
	for _, arg := range call.Args {
		a, ok := arg.(*ast.UnaryExpr)
		if !ok || a.Op != token.AND || s.readOnly[a.Pos()] {
			continue
		}
		key := exprKey(a.X)
		if key == "" {
			continue
		}
		id, isIdent := a.X.(*ast.Ident)
		s.bind(key, isIdent && s.locals[id.Name], a.Pos())
	}
}

// markReadOnlyArgs precomputes the &x arg positions whose control sits under
// a ReadOnly(true) chain: a read-only control never accepts input, so its
// bound local is display-only and exempt.
func (s *funcScan) markReadOnlyArgs(body *ast.BlockStmt) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || callName(call) != "ReadOnly" || len(call.Args) != 1 {
			return true
		}
		if id, ok := call.Args[0].(*ast.Ident); !ok || id.Name != "true" {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		for _, c := range spineCalls(sel.X) {
			if !deferredControl[callName(c)] {
				continue
			}
			for _, arg := range c.Args {
				if u, ok := arg.(*ast.UnaryExpr); ok && u.Op == token.AND {
					s.readOnly[u.Pos()] = true
				}
			}
		}
		return true
	})
}

// spineCalls returns the call chain under e as innermost-first: for
// ui.A().B().C() it yields [ui.A(), .B(), .C()... ] walking Fun.SelectorExpr.X.
func spineCalls(e ast.Expr) []*ast.CallExpr {
	var out []*ast.CallExpr
	for {
		call, ok := e.(*ast.CallExpr)
		if !ok {
			return out
		}
		out = append(out, call)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return out
		}
		e = sel.X
	}
}

// spineLocalBinds returns the names of stack locals bound by deferred
// controls along the receiver chain under e — reads of them inside this
// call's arguments still run during the control's construction chain.
func (s *funcScan) spineLocalBinds(e ast.Expr) []string {
	var out []string
	for _, c := range spineCalls(e) {
		if !deferredControl[callName(c)] {
			continue
		}
		for _, arg := range c.Args {
			u, ok := arg.(*ast.UnaryExpr)
			if !ok || u.Op != token.AND {
				continue
			}
			if id, ok := u.X.(*ast.Ident); ok && s.locals[id.Name] {
				out = append(out, id.Name)
			}
		}
	}
	return out
}

// checkSnapshotCompare implements pattern (b): `before := expr` captured
// before a deferred control bound &expr, then `expr != before` (or ==,
// either operand order) compared before any flushing query.
func (s *funcScan) checkSnapshotCompare(n *ast.BinaryExpr) {
	for _, pair := range [2][2]ast.Expr{{n.X, n.Y}, {n.Y, n.X}} {
		id, ok := pair[0].(*ast.Ident)
		if !ok {
			continue
		}
		snap, ok := s.snaps[id.Name]
		if !ok || snap == "" {
			continue
		}
		if exprKey(pair[1]) != snap {
			continue
		}
		b := s.livePtr(snap)
		if b == nil {
			b = s.liveLocal(snap)
		}
		if b == nil || b.flushed || b.reported || s.flagReads == 0 || s.deferDepth > 0 {
			continue
		}
		b.reported = true
		b.read = true
		s.fail(BoundViolation{
			File: s.file,
			Line: s.fset.Position(n.Pos()).Line,
			Msg: "bound value " + strconv.Quote(snap) + " compared against snapshot " + strconv.Quote(id.Name) +
				" before an input query flushed it: compare sees last frame's value — use the element's Changed() instead",
		})
	}
}

// callName resolves a call's simple name: ui.Checkbox(...) -> "Checkbox",
// f(...) -> "f".
func callName(c *ast.CallExpr) string {
	switch f := c.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

func copyLocals(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// copyBound shallow-copies the binding map for mutually exclusive branches:
// each case clause starts from the bindings live before the switch.
func copyBound(in map[string]*bindingSet) map[string]*bindingSet {
	out := make(map[string]*bindingSet, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
