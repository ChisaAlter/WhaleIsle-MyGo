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
//	a) a stack local (or stack struct field / alias) bound to a deferred
//	   control whose value is read in a decision position before the next
//	   flushing UI query — or never read at all;
//	b) a before-snapshot (`before := expr` or `before := wrap(expr)`)
//	   compared against expr (inline or via helper) after a deferred control
//	   bound &expr without an intervening flushing UI query;
//	c) a bare pointer passed directly (`ui.TextArea(c, p)` or `m.Draft`)
//	   whose pointee `*p` is read in a decision position before flush.
//
// flushCall is every element query in third_party/mygo/ui that runs
// applyInputs() first. Project helpers are never hardcoded here: package-level
// summary analysis infers whether same-package helpers flush unconditionally or
// forward bound parameters into deferred controls.
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
}

// deferredCallback prefixes mark callback registrars whose FuncLit argument
// runs during or after input processing — applyInputs has already written
// every pending binding by then, so bound values read inside are fresh.
func deferredCallback(name string) bool {
	return strings.HasPrefix(name, "On") || strings.HasPrefix(name, "Handle")
}

// deferredControlArgs maps every deferred-writeback control in
// third_party/mygo/ui and internal/ui/coss to the 0-based argument position(s)
// carrying a bound pointer written inside afterInput/onValueInput.
var deferredControlArgs = map[string][]int{
	// ui.* and coss.* controls whose bound pointer is arg 1
	"Autocomplete":          {1},
	"Calendar":              {1},
	"Checkbox":              {1},
	"CheckboxBase":          {1},
	"CollapsibleBase":       {1},
	"ColorPicker":           {1},
	"ColorWell":             {1},
	"Combobox":              {1},
	"ComboboxBase":          {1},
	"DateInput":             {1},
	"EditableText":          {1},
	"NumberInput":           {1},
	"Radio":                 {1},
	"RadioBase":             {1},
	"Rating":                {1},
	"SearchField":           {1},
	"Segmented":             {1},
	"SegmentedBase":         {1},
	"Select":                {1},
	"SelectBase":            {1},
	"Sidebar":               {1},
	"Slider":                {1},
	"SliderBase":            {1},
	"Split":                 {1},
	"SplitVertical":         {1},
	"SplitFraction":         {1},
	"SplitVerticalFraction": {1},
	"StepSlider":            {1},
	"Stepper":               {1},
	"Switch":                {1},
	"SwitchBase":            {1},
	"Tabs":                  {1},
	"TabsBase":              {1},
	"TextArea":              {1},
	"TextAreaBase":          {1},
	"TextAreaBuffer":        {1},
	"TextAreaBufferBase":    {1},
	"TextInput":             {1},
	"TextInputBase":         {1},
	"TextInputBuffer":       {1},
	"TextInputBufferBase":   {1},
	"TimeInput":             {1},
	"Toggle":                {1},
	"ToggleBase":            {1},
	"TokenField":            {1},
	"SegmentedControl":      {1},
	"Input":                 {1},
	"Textarea":              {1},
	"InputGroup":            {1},
	"NumberField":           {1},
	"Composer":              {1},
	"CommandPanel":          {1},
	// ui.* / coss.* controls whose bound pointer is arg 2
	"AccordionItem":  {2},
	"Breadcrumbs":    {2},
	"SidebarSection": {2},
	"CommandPalette": {2},
	// multi-pointer bindings
	"RangeSlider": {1, 2},
	"FindBar":     {1, 2, 4},
}

var knownUIPkgs = map[string]bool{
	"ui":        true,
	"coss":      true,
	"widgets":   true,
	"pagekit":   true,
	"theme":     true,
	"fonttheme": true,
	"design":    true,
	"chrome":    true,
	"i18n":      true,
}

var uiRenderPkgs = map[string]bool{
	"ui":      true,
	"coss":    true,
	"widgets": true,
	"pagekit": true,
	"chrome":  true,
}

var pureDisplayPkgs = map[string]bool{
	"fmt":     true,
	"strings": true,
	"strconv": true,
	"sort":    true,
	"slices":  true,
	"maps":    true,
	"math":    true,
	"time":    true,
}

func controlBoundIndices(call *ast.CallExpr, resolvedName string, sum *pkgSummary) ([]int, bool) {
	if resolvedName == "Collapsible" {
		if sel, ok := unwrapParen(call.Fun).(*ast.SelectorExpr); ok {
			if id, ok := unwrapParen(sel.X).(*ast.Ident); ok && id.Name == "coss" {
				return []int{1}, true
			}
		}
		return []int{2}, true
	}
	if idxs, ok := deferredControlArgs[resolvedName]; ok {
		return idxs, true
	}
	if sum != nil {
		if isPkgFuncCall(call) {
			if params := sum.funcBindParams[resolvedName]; len(params) > 0 {
				return sortedParamKeys(params), true
			}
		} else if params := sum.methodBindParams[resolvedName]; len(params) > 0 {
			return sortedParamKeys(params), true
		}
	}
	return nil, false
}

func sortedParamKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func isPkgFuncCall(call *ast.CallExpr) bool {
	_, ok := unwrapParen(call.Fun).(*ast.Ident)
	return ok
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
	byDir := map[string][]string{}
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			parent := filepath.Dir(path)
			byDir[parent] = append(byDir[parent], path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	var dirKeys []string
	for d := range byDir {
		dirKeys = append(dirKeys, d)
	}
	sort.Strings(dirKeys)

	var out []BoundViolation
	for _, d := range dirKeys {
		paths := byDir[d]
		sort.Strings(paths)
		fset := token.NewFileSet()
		type parsed struct {
			rel string
			ast *ast.File
		}
		var files []parsed
		var asts []*ast.File
		for _, p := range paths {
			f, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				return nil, err
			}
			files = append(files, parsed{rel: filepath.ToSlash(p), ast: f})
			asts = append(asts, f)
		}
		sum := summarizePackage(asts)
		for _, pf := range files {
			out = append(out, scanBoundAST(fset, pf.ast, pf.rel, sum)...)
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
	sum := summarizePackage([]*ast.File{f})
	return scanBoundAST(fset, f, name, sum), nil
}

// pkgSummary holds conservative interprocedural summaries for same-package
// functions and methods: whether a helper flushes unconditionally after any
// deferred control, which pointer params are forwarded into unflushed controls,
// and which snapshot parameters are compared against a struct field inside.
type pkgSummary struct {
	funcFlushes         map[string]bool
	methodFlushes       map[string]bool
	funcBindParams      map[string]map[int]bool
	methodBindParams    map[string]map[int]bool
	funcReadOnlyParam   map[string]map[int]int
	methodReadOnlyParam map[string]map[int]int
	funcSnapshotFields  map[string]map[int]map[string]bool
	methodSnapshotField map[string]map[int]map[string]bool
	funcUIBuilder       map[string]bool
	methodUIBuilder     map[string]bool
	funcReturnsElem     map[string]bool
	methodReturnsElem   map[string]bool
}

type declInfo struct {
	fn           *ast.FuncDecl
	isMethod     bool
	name         string
	paramIndex   map[string]int
	returnsElem  bool
	isUIBuilder  bool
	snapshotUses map[int]map[string]bool
}

func summarizePackage(files []*ast.File) *pkgSummary {
	sum := &pkgSummary{
		funcFlushes:         map[string]bool{},
		methodFlushes:       map[string]bool{},
		funcBindParams:      map[string]map[int]bool{},
		methodBindParams:    map[string]map[int]bool{},
		funcReadOnlyParam:   map[string]map[int]int{},
		methodReadOnlyParam: map[string]map[int]int{},
		funcSnapshotFields:  map[string]map[int]map[string]bool{},
		methodSnapshotField: map[string]map[int]map[string]bool{},
		funcUIBuilder:       map[string]bool{},
		methodUIBuilder:     map[string]bool{},
		funcReturnsElem:     map[string]bool{},
		methodReturnsElem:   map[string]bool{},
	}
	var decls []declInfo
	methodFlushAll := map[string]bool{}
	methodSeen := map[string]bool{}
	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			isMethod := fn.Recv != nil
			params := paramIndexMap(fn)
			retElem := returnMentionsUIElem(fn)
			uiBuilder := paramsMentionUI(fn) || bodyCallsUI(fn.Body)
			snaps := extractParamSnapshotFields(fn.Body, params)
			decls = append(decls, declInfo{
				fn:           fn,
				isMethod:     isMethod,
				name:         fn.Name.Name,
				paramIndex:   params,
				returnsElem:  retElem,
				isUIBuilder:  uiBuilder,
				snapshotUses: snaps,
			})
			if isMethod {
				if uiBuilder {
					sum.methodUIBuilder[fn.Name.Name] = true
				}
				if retElem {
					sum.methodReturnsElem[fn.Name.Name] = true
				}
				for idx, fs := range snaps {
					if sum.methodSnapshotField[fn.Name.Name] == nil {
						sum.methodSnapshotField[fn.Name.Name] = map[int]map[string]bool{}
					}
					if sum.methodSnapshotField[fn.Name.Name][idx] == nil {
						sum.methodSnapshotField[fn.Name.Name][idx] = map[string]bool{}
					}
					for field := range fs {
						sum.methodSnapshotField[fn.Name.Name][idx][field] = true
					}
				}
			} else {
				if uiBuilder {
					sum.funcUIBuilder[fn.Name.Name] = true
				}
				if retElem {
					sum.funcReturnsElem[fn.Name.Name] = true
				}
				if len(snaps) > 0 {
					sum.funcSnapshotFields[fn.Name.Name] = snaps
				}
			}
		}
	}

	// Fixed-point computation for flushes and deferredBindParams.
	for changed := true; changed; {
		changed = false
		nextMethodFlush := map[string]bool{}
		for k := range methodFlushAll {
			delete(methodFlushAll, k)
		}
		for k := range methodSeen {
			delete(methodSeen, k)
		}
		for _, d := range decls {
			flushes, bindParams, roParams := analyzeFuncSummary(d, sum)
			if d.isMethod {
				if !methodSeen[d.name] {
					methodSeen[d.name] = true
					nextMethodFlush[d.name] = flushes
				} else if !flushes {
					nextMethodFlush[d.name] = false
				}
				if !flushes {
					for idx := range bindParams {
						if sum.methodBindParams[d.name] == nil {
							sum.methodBindParams[d.name] = map[int]bool{}
						}
						if !sum.methodBindParams[d.name][idx] {
							sum.methodBindParams[d.name][idx] = true
							changed = true
						}
					}
					for bIdx, rIdx := range roParams {
						if sum.methodReadOnlyParam[d.name] == nil {
							sum.methodReadOnlyParam[d.name] = map[int]int{}
						}
						if cur, ok := sum.methodReadOnlyParam[d.name][bIdx]; !ok || cur != rIdx {
							sum.methodReadOnlyParam[d.name][bIdx] = rIdx
							changed = true
						}
					}
				}
			} else {
				if flushes && !sum.funcFlushes[d.name] {
					sum.funcFlushes[d.name] = true
					changed = true
				}
				if !flushes {
					for idx := range bindParams {
						if sum.funcBindParams[d.name] == nil {
							sum.funcBindParams[d.name] = map[int]bool{}
						}
						if !sum.funcBindParams[d.name][idx] {
							sum.funcBindParams[d.name][idx] = true
							changed = true
						}
					}
					for bIdx, rIdx := range roParams {
						if sum.funcReadOnlyParam[d.name] == nil {
							sum.funcReadOnlyParam[d.name] = map[int]int{}
						}
						if cur, ok := sum.funcReadOnlyParam[d.name][bIdx]; !ok || cur != rIdx {
							sum.funcReadOnlyParam[d.name][bIdx] = rIdx
							changed = true
						}
					}
				} else {
					if len(sum.funcBindParams[d.name]) > 0 {
						delete(sum.funcBindParams, d.name)
						changed = true
					}
					if len(sum.funcReadOnlyParam[d.name]) > 0 {
						delete(sum.funcReadOnlyParam, d.name)
						changed = true
					}
				}
			}
		}
		for name, fl := range nextMethodFlush {
			if fl != sum.methodFlushes[name] {
				sum.methodFlushes[name] = fl
				changed = true
			}
			if fl {
				if len(sum.methodBindParams[name]) > 0 {
					delete(sum.methodBindParams, name)
					changed = true
				}
				if len(sum.methodReadOnlyParam[name]) > 0 {
					delete(sum.methodReadOnlyParam, name)
					changed = true
				}
			}
		}
		for _, d := range decls {
			if d.isMethod {
				if !sum.methodUIBuilder[d.name] && bodyCallsSummaryUI(d.fn.Body, sum) {
					sum.methodUIBuilder[d.name] = true
					changed = true
				}
			} else {
				if !sum.funcUIBuilder[d.name] && bodyCallsSummaryUI(d.fn.Body, sum) {
					sum.funcUIBuilder[d.name] = true
					changed = true
				}
			}
		}
	}
	return sum
}

func paramIndexMap(fn *ast.FuncDecl) map[string]int {
	out := map[string]int{}
	if fn.Type.Params == nil {
		return out
	}
	idx := 0
	for _, field := range fn.Type.Params.List {
		if len(field.Names) == 0 {
			idx++
			continue
		}
		for _, nm := range field.Names {
			out[nm.Name] = idx
			idx++
		}
	}
	return out
}

func returnMentionsUIElem(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}
	found := false
	for _, res := range fn.Type.Results.List {
		ast.Inspect(res.Type, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if id.Name == "Element" || strings.HasSuffix(id.Name, "Parts") {
					found = true
				}
			}
			return !found
		})
	}
	return found
}

func paramsMentionUI(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}
	found := false
	for _, field := range fn.Type.Params.List {
		ast.Inspect(field.Type, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if id.Name == "Context" || id.Name == "Element" || id.Name == "Render" || strings.HasSuffix(id.Name, "Parts") {
					found = true
				}
			}
			return !found
		})
	}
	return found
}

func bodyCallsUI(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := unwrapParen(call.Fun).(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "Key" {
				found = true
				return false
			}
			if id, ok := unwrapParen(sel.X).(*ast.Ident); ok && uiRenderPkgs[id.Name] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func bodyCallsSummaryUI(body *ast.BlockStmt, sum *pkgSummary) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := unwrapParen(call.Fun).(type) {
		case *ast.Ident:
			if sum.funcUIBuilder[fun.Name] {
				found = true
				return false
			}
		case *ast.SelectorExpr:
			if sum.methodUIBuilder[fun.Sel.Name] {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func extractParamSnapshotFields(body *ast.BlockStmt, params map[string]int) map[int]map[string]bool {
	out := map[int]map[string]bool{}
	if len(params) == 0 {
		return out
	}
	ast.Inspect(body, func(n ast.Node) bool {
		bin, ok := n.(*ast.BinaryExpr)
		if !ok || !isCompareOp(bin.Op) {
			return true
		}
		for _, pair := range [2][2]ast.Expr{{bin.X, bin.Y}, {bin.Y, bin.X}} {
			id, ok := unwrapParen(pair[0]).(*ast.Ident)
			if !ok {
				continue
			}
			pIdx, isParam := params[id.Name]
			if !isParam {
				continue
			}
			ast.Inspect(pair[1], func(sub ast.Node) bool {
				if sel, ok := sub.(*ast.SelectorExpr); ok {
					if out[pIdx] == nil {
						out[pIdx] = map[string]bool{}
					}
					out[pIdx][sel.Sel.Name] = true
				}
				return true
			})
		}
		return true
	})
	return out
}

// analyzeFuncSummary walks d.fn unconditionally (through blocks, statement
// expressions, IfStmt.Cond LHS, synchronous non-deferred FuncLit args, and
// ReturnStmt expressions) to decide:
//   - flushes: whether the function always finishes with a flushing UI query
//     after any deferred control call;
//   - bindParams: parameter indices forwarded as bound pointers into unflushed
//     deferred controls;
//   - readOnlyParam: bound parameter index -> boolean parameter index whose
//     ReadOnly(roParam) call disables edits on that bound parameter.
func analyzeFuncSummary(d declInfo, sum *pkgSummary) (bool, map[int]bool, map[int]int) {
	aliases := map[string]int{}
	for nm, idx := range d.paramIndex {
		aliases[nm] = idx
	}
	uiVars := collectUIVarsFromParams(d.fn)
	bindParams := map[int]bool{}
	readOnlyParam := map[int]int{}
	elemParamBinds := map[string][]int{}
	hasFlush := false
	hasUnflushedCtrl := false
	deferDepth := 0
	condDepth := 0

	var walkExpr func(ast.Expr)
	var walkBlock func(*ast.BlockStmt)
	var walkStmt func(ast.Stmt)

	isUIRecv := func(e ast.Expr) bool {
		return isUIReceiverExpr(e, uiVars, sum)
	}

	paramOfExpr := func(arg ast.Expr) (int, bool) {
		id, ok := unwrapParen(arg).(*ast.Ident)
		if !ok {
			return 0, false
		}
		pIdx, ok := aliases[id.Name]
		return pIdx, ok
	}

	boundParamOfArg := func(arg ast.Expr) (int, bool) {
		u := unwrapParen(arg)
		if un, ok := u.(*ast.UnaryExpr); ok && un.Op == token.AND {
			u = unwrapParen(un.X)
		}
		return paramOfExpr(u)
	}

	boundParamIndicesInExpr := func(e ast.Expr) []int {
		var out []int
		for _, c := range spineCalls(e) {
			cname := callName(c)
			idxs, ok := controlBoundIndices(c, cname, sum)
			if !ok {
				continue
			}
			roMap := summaryReadOnlyMap(c, cname, sum)
			for _, idx := range idxs {
				if idx < 0 || idx >= len(c.Args) {
					continue
				}
				if roIdx, hasRO := roMap[idx]; hasRO && roIdx >= 0 && roIdx < len(c.Args) && isLiteralTrueExpr(c.Args[roIdx]) {
					continue
				}
				if pIdx, ok := boundParamOfArg(c.Args[idx]); ok {
					out = append(out, pIdx)
				}
			}
		}
		cur := unwrapParen(e)
		for {
			switch v := cur.(type) {
			case *ast.Ident:
				if binds := elemParamBinds[v.Name]; len(binds) > 0 {
					out = append(out, binds...)
				}
				return out
			case *ast.SelectorExpr:
				cur = unwrapParen(v.X)
			case *ast.CallExpr:
				if sel, ok := unwrapParen(v.Fun).(*ast.SelectorExpr); ok {
					cur = unwrapParen(sel.X)
				} else {
					return out
				}
			default:
				return out
			}
		}
	}

	recordBoundExpr := func(arg ast.Expr) {
		if pIdx, ok := boundParamOfArg(arg); ok {
			bindParams[pIdx] = true
		}
	}

	walkCallArgs := func(call *ast.CallExpr, fnName string) {
		var s *funcScan // not used; walk args linearly
		_ = s
		if sel, ok := unwrapParen(call.Fun).(*ast.SelectorExpr); ok {
			walkExpr(sel.X)
		} else {
			walkExpr(call.Fun)
		}
		for _, arg := range call.Args {
			isLit := false
			if _, ok := unwrapParen(arg).(*ast.FuncLit); ok {
				isLit = true
			}
			if isLit && deferredCallback(fnName) {
				deferDepth++
				walkExpr(arg)
				deferDepth--
			} else {
				walkExpr(arg)
			}
		}
	}

	walkExpr = func(e ast.Expr) {
		if e == nil {
			return
		}
		switch n := e.(type) {
		case *ast.CallExpr:
			fnName := callName(n)
			walkCallArgs(n, fnName)
			if deferDepth > 0 {
				return
			}
			if condDepth == 0 {
				if fnName == "ReadOnly" && len(n.Args) == 1 {
					if sel, ok := unwrapParen(n.Fun).(*ast.SelectorExpr); ok {
						if roIdx, ok := paramOfExpr(n.Args[0]); ok {
							for _, bIdx := range boundParamIndicesInExpr(sel.X) {
								readOnlyParam[bIdx] = roIdx
							}
						}
					}
				}
				if roMap := summaryReadOnlyMap(n, fnName, sum); len(roMap) > 0 {
					for bArgIdx, rArgIdx := range roMap {
						if bArgIdx >= 0 && bArgIdx < len(n.Args) && rArgIdx >= 0 && rArgIdx < len(n.Args) {
							if bIdx, ok1 := boundParamOfArg(n.Args[bArgIdx]); ok1 {
								if rIdx, ok2 := paramOfExpr(n.Args[rArgIdx]); ok2 {
									readOnlyParam[bIdx] = rIdx
								}
							}
						}
					}
				}
			}
			if idxs, ok := controlBoundIndices(n, fnName, sum); ok {
				roMap := summaryReadOnlyMap(n, fnName, sum)
				boundAny := false
				for _, idx := range idxs {
					if idx < 0 || idx >= len(n.Args) {
						continue
					}
					if roIdx, hasRO := roMap[idx]; hasRO && roIdx >= 0 && roIdx < len(n.Args) && isLiteralTrueExpr(n.Args[roIdx]) {
						continue
					}
					boundAny = true
					recordBoundExpr(n.Args[idx])
				}
				if boundAny {
					hasUnflushedCtrl = true
				}
			} else if isSummaryFlushCall(n, fnName, isUIRecv, sum) {
				hasFlush = true
				hasUnflushedCtrl = false
			}
		case *ast.BinaryExpr:
			walkExpr(n.X)
			if n.Op != token.LAND && n.Op != token.LOR {
				walkExpr(n.Y)
			}
		case *ast.UnaryExpr:
			walkExpr(n.X)
		case *ast.ParenExpr:
			walkExpr(n.X)
		case *ast.SelectorExpr:
			walkExpr(n.X)
		case *ast.IndexExpr:
			walkExpr(n.X)
			walkExpr(n.Index)
		case *ast.IndexListExpr:
			walkExpr(n.X)
			for _, idx := range n.Indices {
				walkExpr(idx)
			}
		case *ast.SliceExpr:
			walkExpr(n.X)
			walkExpr(n.Low)
			walkExpr(n.High)
			walkExpr(n.Max)
		case *ast.StarExpr:
			walkExpr(n.X)
		case *ast.TypeAssertExpr:
			walkExpr(n.X)
		case *ast.FuncLit:
			if deferDepth == 0 {
				walkBlock(n.Body)
			}
		case *ast.CompositeLit:
			for _, elt := range n.Elts {
				walkExpr(elt)
			}
		case *ast.KeyValueExpr:
			saved := deferDepth
			if id, ok := n.Key.(*ast.Ident); ok && deferredCallback(id.Name) {
				deferDepth++
			}
			walkExpr(n.Value)
			deferDepth = saved
		}
	}

	walkStmt = func(st ast.Stmt) {
		switch n := st.(type) {
		case *ast.AssignStmt:
			for _, rhs := range n.Rhs {
				if _, isLit := unwrapParen(rhs).(*ast.FuncLit); !isLit {
					walkExpr(rhs)
				}
			}
			if len(n.Rhs) == len(n.Lhs) {
				for i, lhs := range n.Lhs {
					id, ok := lhs.(*ast.Ident)
					if !ok {
						continue
					}
					if rhsID, ok := unwrapParen(n.Rhs[i]).(*ast.Ident); ok {
						if pIdx, isAlias := aliases[rhsID.Name]; isAlias {
							aliases[id.Name] = pIdx
						}
					}
					if isUIReceiverExpr(n.Rhs[i], uiVars, sum) {
						uiVars[id.Name] = true
					}
					if binds := boundParamIndicesInExpr(n.Rhs[i]); len(binds) > 0 {
						elemParamBinds[id.Name] = binds
					} else {
						delete(elemParamBinds, id.Name)
					}
				}
			}
		case *ast.DeclStmt:
			if gd, ok := n.Decl.(*ast.GenDecl); ok {
				for _, spec := range gd.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for i, val := range vs.Values {
							if _, isLit := unwrapParen(val).(*ast.FuncLit); !isLit {
								walkExpr(val)
							}
							if i < len(vs.Names) {
								if isUIReceiverExpr(val, uiVars, sum) {
									uiVars[vs.Names[i].Name] = true
								}
								if binds := boundParamIndicesInExpr(val); len(binds) > 0 {
									elemParamBinds[vs.Names[i].Name] = binds
								} else {
									delete(elemParamBinds, vs.Names[i].Name)
								}
							}
						}
					}
				}
			}
		case *ast.ExprStmt:
			walkExpr(n.X)
		case *ast.ReturnStmt:
			for _, r := range n.Results {
				walkExpr(r)
			}
		case *ast.IfStmt:
			if n.Init != nil {
				walkStmt(n.Init)
			}
			// Cond LHS evaluates unconditionally (e.g. if !ui.Select(...).Changed()).
			walkExpr(n.Cond)
			// Only scan Body/Else for bindParams, without letting conditional flushes mark flushes=true.
			savedFlush, savedUnflushed := hasFlush, hasUnflushedCtrl
			condDepth++
			walkBlock(n.Body)
			if n.Else != nil {
				switch els := n.Else.(type) {
				case *ast.BlockStmt:
					walkBlock(els)
				case *ast.IfStmt:
					walkStmt(els)
				}
			}
			condDepth--
			hasFlush = savedFlush
			hasUnflushedCtrl = savedUnflushed || (hasUnflushedCtrl && !savedFlush)
		case *ast.LabeledStmt:
			walkStmt(n.Stmt)
		case *ast.BlockStmt:
			walkBlock(n)
		}
	}

	walkBlock = func(b *ast.BlockStmt) {
		if b == nil {
			return
		}
		for _, st := range b.List {
			walkStmt(st)
		}
	}

	walkBlock(d.fn.Body)
	return hasFlush && !hasUnflushedCtrl, bindParams, readOnlyParam
}

func isLiteralTrueExpr(e ast.Expr) bool {
	id, ok := unwrapParen(e).(*ast.Ident)
	return ok && id.Name == "true"
}

func summaryReadOnlyMap(call *ast.CallExpr, resolvedName string, sum *pkgSummary) map[int]int {
	if sum == nil {
		return nil
	}
	if isPkgFuncCall(call) {
		return sum.funcReadOnlyParam[resolvedName]
	}
	return sum.methodReadOnlyParam[resolvedName]
}

func collectUIVarsFromParams(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if fn.Type.Params == nil {
		return out
	}
	for _, field := range fn.Type.Params.List {
		mentionsUI := false
		ast.Inspect(field.Type, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if id.Name == "Context" || id.Name == "Element" || id.Name == "Render" || strings.HasSuffix(id.Name, "Parts") {
					mentionsUI = true
				}
			}
			return !mentionsUI
		})
		if mentionsUI {
			for _, nm := range field.Names {
				out[nm.Name] = true
			}
		}
	}
	return out
}

func isSummaryFlushCall(n *ast.CallExpr, name string, isUIRecv func(ast.Expr) bool, sum *pkgSummary) bool {
	if isPkgFuncCall(n) {
		return sum != nil && sum.funcFlushes[name]
	}
	if sum != nil && sum.methodFlushes[name] {
		return true
	}
	if flushCall[name] {
		if sel, ok := unwrapParen(n.Fun).(*ast.SelectorExpr); ok {
			return isUIRecv(sel.X)
		}
	}
	return false
}

func isUIResultCall(call *ast.CallExpr, sum *pkgSummary) bool {
	name := callName(call)
	if _, ok := deferredControlArgs[name]; ok || name == "Collapsible" {
		return true
	}
	switch fun := unwrapParen(call.Fun).(type) {
	case *ast.Ident:
		return sum != nil && sum.funcReturnsElem[fun.Name]
	case *ast.SelectorExpr:
		if pkg, ok := unwrapParen(fun.X).(*ast.Ident); ok && knownUIPkgs[pkg.Name] {
			return true
		}
		if recvCall, ok := unwrapParen(fun.X).(*ast.CallExpr); ok {
			return isUIResultCall(recvCall, sum)
		}
		return sum != nil && sum.methodReturnsElem[fun.Sel.Name]
	}
	return false
}

func isUIReceiverExpr(e ast.Expr, uiVars map[string]bool, sum *pkgSummary) bool {
	e = unwrapParen(e)
	switch v := e.(type) {
	case *ast.Ident:
		return uiVars[v.Name]
	case *ast.CallExpr:
		if isUIResultCall(v, sum) {
			return true
		}
		if sel, ok := unwrapParen(v.Fun).(*ast.SelectorExpr); ok {
			return isUIReceiverExpr(sel.X, uiVars, sum)
		}
		return false
	case *ast.SelectorExpr:
		if id, ok := unwrapParen(v.X).(*ast.Ident); ok && uiVars[id.Name] {
			return true
		}
		return isUIReceiverExpr(v.X, uiVars, sum)
	}
	return false
}

func unwrapParen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

func isCompareOp(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
		return true
	}
	return false
}

// exprKey renders an identifier, selector chain ("x", "s.Field"), or indexed
// target ("s.Items[i]") with outer parens stripped. Anything else returns "".
func exprKey(e ast.Expr) string {
	switch e := unwrapParen(e).(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if base := exprKey(e.X); base != "" {
			return base + "." + e.Sel.Name
		}
	case *ast.IndexExpr:
		base := exprKey(e.X)
		if base == "" {
			return ""
		}
		idx := simpleIndexKey(e.Index)
		if idx == "" {
			return ""
		}
		return base + "[" + idx + "]"
	}
	return ""
}

func simpleIndexKey(e ast.Expr) string {
	switch v := unwrapParen(e).(type) {
	case *ast.BasicLit:
		return v.Value
	default:
		return exprKey(v)
	}
}

func exprCanon(e ast.Expr) string {
	switch v := unwrapParen(e).(type) {
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		return v.Value
	case *ast.SelectorExpr:
		if b := exprCanon(v.X); b != "" {
			return b + "." + v.Sel.Name
		}
	case *ast.IndexExpr:
		if b, idx := exprCanon(v.X), exprCanon(v.Index); b != "" && idx != "" {
			return b + "[" + idx + "]"
		}
	case *ast.UnaryExpr:
		if x := exprCanon(v.X); x != "" {
			return v.Op.String() + x
		}
	case *ast.StarExpr:
		if x := exprCanon(v.X); x != "" {
			return "*" + x
		}
	case *ast.CallExpr:
		fn := exprCanon(v.Fun)
		if fn == "" {
			return ""
		}
		args := make([]string, len(v.Args))
		for i, a := range v.Args {
			ac := exprCanon(a)
			if ac == "" {
				return ""
			}
			args[i] = ac
		}
		return fn + "(" + strings.Join(args, ",") + ")"
	}
	return ""
}

func exprRootName(key string) string {
	if i := strings.IndexAny(key, ".["); i >= 0 {
		return key[:i]
	}
	return key
}

func leafFieldName(key string) string {
	if i := strings.LastIndexByte(key, '.'); i >= 0 {
		return key[i+1:]
	}
	return key
}

// binding records one deferred writeback onto the named expression.
type binding struct {
	key     string // exprKey of the bound value ("x", "cfg.On", "s.Title", "m.Draft")
	local   bool   // bound via stack-owned storage whose reads must see fresh writes
	pos     token.Pos
	flushed bool
	read    bool // value observed after the binding
	// reported guards against double-flagging one binding.
	reported bool
}

// bindingSet keeps stack-local bindings and pointer/field bindings in separate
// lists so a bare pointer argument never shadows an &x binding for the same
// identifier.
type bindingSet struct {
	local []*binding
	ptr   []*binding
}

func (bs *bindingSet) all() []*binding { return append(append([]*binding{}, bs.local...), bs.ptr...) }

type snapInfo struct {
	root  string // bound exprKey (e.g. "s.T")
	canon string // full canonical snapshot expr (e.g. "strings.TrimSpace(s.T)")
}

type ptrTarget struct {
	key   string
	local bool
}

// funcScan walks one top-level function in source order. Nested FuncLit
// bodies share the stream: mygo runs Children/style closures during the
// enclosing call, so source order is execution order for the build pass.
type funcScan struct {
	fset         *token.FileSet
	file         string
	pkg          *pkgSummary
	locals       map[string]bool        // declared stack variables / params
	valueLocals  map[string]bool        // non-pointer stack locals/value params whose fields are stack-owned
	fnAliases    map[string]string      // fn := ui.Checkbox -> "Checkbox"
	ptrAliases   map[string]ptrTarget   // p := &v -> target{key: "v", local: true}
	uiElemVars   map[string]bool        // vars holding UI context / Element / Parts
	bound        map[string]*bindingSet // live bindings per bound expression key
	snaps        map[string]snapInfo    // var name -> snapshot metadata
	readOnly     map[token.Pos]bool     // bound arg positions under a ReadOnly(true) chain
	exempt       map[string]int         // bound locals readable in this context
	exemptAll    int                    // inside a built-element chain's config arg
	elemVars     map[string][]string    // vars holding a bound element -> bound local names
	deferArg     bool                   // current arg runs after input processing
	deferDepth   int                    // inside a deferred-callback closure
	flagReads    int                    // reads in decision positions (conds, assigns, calls)
	callArgReads int                    // reads in non-UI statement call arguments
	out          *[]BoundViolation
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

func (s *funcScan) liveLocal(key string) *binding {
	if bs := s.bound[key]; bs != nil && len(bs.local) > 0 {
		return bs.local[len(bs.local)-1]
	}
	return nil
}

func (s *funcScan) livePtr(key string) *binding {
	if bs := s.bound[key]; bs != nil && len(bs.ptr) > 0 {
		return bs.ptr[len(bs.ptr)-1]
	}
	return nil
}

func (s *funcScan) liveBinding(key string) *binding {
	if b := s.livePtr(key); b != nil {
		return b
	}
	return s.liveLocal(key)
}

func (s *funcScan) flush() {
	for _, bs := range s.bound {
		for _, b := range bs.all() {
			b.flushed = true
		}
	}
}

func (s *funcScan) snapshotFlushed() map[*binding]bool {
	snap := map[*binding]bool{}
	for _, bs := range s.bound {
		for _, b := range bs.all() {
			snap[b] = b.flushed
		}
	}
	return snap
}

func (s *funcScan) restoreFlushed(snap map[*binding]bool) {
	for _, bs := range s.bound {
		for _, b := range bs.all() {
			if was, ok := snap[b]; ok {
				b.flushed = was
			}
		}
	}
}

func intersectFlushedMaps(a, b map[*binding]bool) map[*binding]bool {
	out := make(map[*binding]bool, len(a)+len(b))
	for k, va := range a {
		if vb, ok := b[k]; ok {
			out[k] = va && vb
		} else {
			out[k] = va
		}
	}
	for k, vb := range b {
		if _, ok := a[k]; !ok {
			out[k] = vb
		}
	}
	return out
}

func (s *funcScan) intersectFlushed(a, b map[*binding]bool) {
	s.restoreFlushed(intersectFlushedMaps(a, b))
}

// unbind drops bindings rooted at key (both "cfg" and "cfg.On") when key is
// redeclared or assigned over.
func (s *funcScan) unbind(key string) {
	delete(s.bound, key)
	dot := key + "."
	brack := key + "["
	for k := range s.bound {
		if strings.HasPrefix(k, dot) || strings.HasPrefix(k, brack) {
			delete(s.bound, k)
		}
	}
}

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

// readLocal observes one use of a bound stack-local expression ("x" or "cfg.On").
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

// finish flags bound stack locals nobody ever read or queried.
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

func scanBoundAST(fset *token.FileSet, f *ast.File, file string, pkg *pkgSummary) []BoundViolation {
	var out []BoundViolation
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		s := &funcScan{
			fset:        fset,
			file:        file,
			pkg:         pkg,
			locals:      map[string]bool{},
			valueLocals: map[string]bool{},
			fnAliases:   map[string]string{},
			ptrAliases:  map[string]ptrTarget{},
			uiElemVars:  collectUIVarsFromParams(fn),
			bound:       map[string]*bindingSet{},
			snaps:       map[string]snapInfo{},
			readOnly:    map[token.Pos]bool{},
			exempt:      map[string]int{},
			elemVars:    map[string][]string{},
			out:         &out,
		}
		// Parameters are stack locals too: &param writes the caller-passed copy.
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				valType := isValueStructType(field.Type)
				for _, name := range field.Names {
					s.locals[name.Name] = true
					if valType {
						s.valueLocals[name.Name] = true
					}
				}
			}
		}
		s.markReadOnlyArgs(fn.Body)
		s.walkBlock(fn.Body)
		s.finish()
	}
	return out
}

func isValueStructType(typ ast.Expr) bool {
	if typ == nil {
		return false
	}
	switch typ.(type) {
	case *ast.StarExpr, *ast.InterfaceType, *ast.MapType, *ast.ArrayType, *ast.ChanType, *ast.FuncType:
		return false
	}
	return true
}

func (s *funcScan) walkBlock(b *ast.BlockStmt) {
	for _, stmt := range b.List {
		s.walkStmt(stmt)
	}
}

func (s *funcScan) recordSnapshotAndAlias(idName string, rhs ast.Expr, isDefine bool) {
	u := unwrapParen(rhs)
	// Record function aliases: cb := ui.Checkbox.
	if alias := s.resolveFuncRef(u); alias != "" {
		s.fnAliases[idName] = alias
	} else {
		delete(s.fnAliases, idName)
	}
	// Record pointer aliases: p := &v or p := q (where q already aliases &v).
	if pt, ok := s.resolvePtrTarget(u); ok {
		s.ptrAliases[idName] = pt
	} else {
		delete(s.ptrAliases, idName)
	}
	// Track value structs constructed inline: cfg := Config{...}.
	if isDefine {
		if _, isComp := u.(*ast.CompositeLit); isComp {
			s.valueLocals[idName] = true
		}
	}
	// Track UI Element variables.
	if isUIReceiverExpr(u, s.uiElemVars, s.pkg) {
		s.uiElemVars[idName] = true
	}
	// Track before-snapshots: direct (`before := s.T`) or pure-wrapper (`before := strings.TrimSpace(s.T)`).
	if info, ok := s.extractSnapInfo(u); ok {
		s.snaps[idName] = info
	} else {
		delete(s.snaps, idName)
	}
	// v := <control call> remembers which bound values v's element owns.
	if binds := s.spineLocalBinds(u); len(binds) > 0 {
		s.elemVars[idName] = binds
	} else {
		delete(s.elemVars, idName)
	}
}

func (s *funcScan) resolveFuncRef(e ast.Expr) string {
	switch v := unwrapParen(e).(type) {
	case *ast.Ident:
		if target, ok := s.fnAliases[v.Name]; ok {
			return target
		}
		if _, ok := deferredControlArgs[v.Name]; ok || v.Name == "Collapsible" || flushCall[v.Name] {
			return v.Name
		}
	case *ast.SelectorExpr:
		name := v.Sel.Name
		if _, ok := deferredControlArgs[name]; ok || name == "Collapsible" || flushCall[name] {
			return name
		}
	}
	return ""
}

func (s *funcScan) resolvePtrTarget(e ast.Expr) (ptrTarget, bool) {
	switch v := unwrapParen(e).(type) {
	case *ast.UnaryExpr:
		if v.Op == token.AND {
			if key := exprKey(v.X); key != "" {
				return ptrTarget{key: key, local: s.isStackOwnedExpr(v.X, key)}, true
			}
		}
	case *ast.Ident:
		if pt, ok := s.ptrAliases[v.Name]; ok {
			return pt, true
		}
	}
	return ptrTarget{}, false
}

func (s *funcScan) isStackOwnedExpr(e ast.Expr, key string) bool {
	u := unwrapParen(e)
	if id, ok := u.(*ast.Ident); ok {
		return s.locals[id.Name]
	}
	if _, ok := u.(*ast.SelectorExpr); ok {
		root := exprRootName(key)
		return s.valueLocals[root]
	}
	return false
}

func (s *funcScan) extractSnapInfo(e ast.Expr) (snapInfo, bool) {
	if key := exprKey(e); key != "" {
		return snapInfo{root: key, canon: key}, true
	}
	if star, ok := unwrapParen(e).(*ast.StarExpr); ok {
		if key := exprKey(star.X); key != "" {
			return snapInfo{root: key, canon: "*" + key}, true
		}
	}
	call, ok := unwrapParen(e).(*ast.CallExpr)
	if !ok || s.isUIFlushCall(call) {
		return snapInfo{}, false
	}
	if _, isCtrl := controlBoundIndices(call, s.resolvedCallName(call), s.pkg); isCtrl {
		return snapInfo{}, false
	}
	canon := exprCanon(call)
	if canon == "" {
		return snapInfo{}, false
	}
	// If the pure wrapper closes over one live or candidate expression (e.g. strings.TrimSpace(s.T)),
	// associate the snapshot with that bound expression root.
	var found string
	multi := false
	for _, arg := range call.Args {
		if k := s.findBoundKeyInExpr(arg); k != "" {
			if found == "" {
				found = k
			} else if found != k {
				multi = true
			}
		}
	}
	if found != "" && !multi {
		return snapInfo{root: found, canon: canon}, true
	}
	return snapInfo{}, false
}

func (s *funcScan) findBoundKeyInExpr(e ast.Expr) string {
	if k := exprKey(e); k != "" {
		return k
	}
	switch v := unwrapParen(e).(type) {
	case *ast.StarExpr:
		return exprKey(v.X)
	case *ast.CallExpr:
		for _, a := range v.Args {
			if k := s.findBoundKeyInExpr(a); k != "" {
				return k
			}
		}
	}
	return ""
}

func (s *funcScan) walkStmt(n ast.Stmt) {
	switch n := n.(type) {
	case *ast.AssignStmt:
		// Evaluate RHS before recording LHS bindings, matching Go order.
		for _, e := range n.Rhs {
			saved := s.deferArg
			if _, ok := unwrapParen(e).(*ast.FuncLit); ok {
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
					s.recordSnapshotAndAlias(id.Name, n.Rhs[i], n.Tok == token.DEFINE)
				}
				if n.Tok == token.ASSIGN {
					s.unbind(id.Name)
				}
				continue
			}
			// s.F = v / s[i] = v: writes, not stale reads — walk for calls without
			// flagging the target base itself, while still catching rhs reads already walked.
			s.walkExpr(lhs)
		}
	case *ast.DeclStmt:
		if d, ok := n.Decl.(*ast.GenDecl); ok {
			for _, spec := range d.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					valueStruct := isValueStructType(vs.Type)
					for _, e := range vs.Values {
						saved := s.deferArg
						if _, ok := unwrapParen(e).(*ast.FuncLit); ok {
							s.deferArg = true
						}
						s.flagReads++
						s.walkExpr(e)
						s.flagReads--
						s.deferArg = saved
					}
					for i, name := range vs.Names {
						s.locals[name.Name] = true
						if valueStruct {
							s.valueLocals[name.Name] = true
						}
						s.unbind(name.Name)
						if i < len(vs.Values) {
							s.recordSnapshotAndAlias(name.Name, vs.Values[i], true)
						} else {
							delete(s.snaps, name.Name)
						}
					}
				}
			}
		}
	case *ast.ExprStmt:
		if call, ok := unwrapParen(n.X).(*ast.CallExpr); ok && !s.isUICall(call) {
			s.walkCallWithDecisionArgs(call)
			return
		}
		s.walkExpr(n.X)
	case *ast.ReturnStmt:
		// Returning a bound value without flushing (e.g. `return size` after
		// Split) returns last frame's value and is flagged; return statements that
		// invoke a flushing UI query (like `return selected, trigger.Changed() && ...`)
		// follow the historyTurnPicker idiom and stay allowed.
		hasFlush := s.exprsContainFlush(n.Results)
		if !hasFlush {
			s.flagReads++
		}
		for _, e := range n.Results {
			s.walkExpr(e)
		}
		if !hasFlush {
			s.flagReads--
		}
	case *ast.IfStmt:
		if n.Init != nil {
			s.walkStmt(n.Init)
		}
		trueFl, falseFl := s.walkCond(n.Cond)
		savedPtrs := copyPtrMap(s.ptrAliases)
		s.restoreFlushed(trueFl)
		s.walkBlock(n.Body)
		afterBody := s.snapshotFlushed()
		s.ptrAliases = copyPtrMap(savedPtrs)
		afterElse := falseFl
		if n.Else != nil {
			s.restoreFlushed(falseFl)
			switch els := n.Else.(type) {
			case *ast.BlockStmt:
				s.walkBlock(els)
			case *ast.IfStmt:
				s.walkStmt(els)
			}
			afterElse = s.snapshotFlushed()
			s.ptrAliases = savedPtrs
		} else {
			s.ptrAliases = savedPtrs
		}
		s.intersectFlushed(afterBody, afterElse)
	case *ast.SwitchStmt:
		if n.Init != nil {
			s.walkStmt(n.Init)
		}
		if n.Tag != nil {
			s.flagReads++
			s.walkExpr(n.Tag)
			s.flagReads--
		}
		pre := s.snapshotFlushed()
		saved := s.bound
		savedPtrs := copyPtrMap(s.ptrAliases)
		for _, stmt := range n.Body.List {
			s.bound = copyBound(saved)
			s.ptrAliases = copyPtrMap(savedPtrs)
			s.restoreFlushed(pre)
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
		s.ptrAliases = savedPtrs
		s.restoreFlushed(pre)
	case *ast.TypeSwitchStmt:
		if n.Init != nil {
			s.walkStmt(n.Init)
		}
		s.walkStmt(n.Assign)
		pre := s.snapshotFlushed()
		saved := s.bound
		savedPtrs := copyPtrMap(s.ptrAliases)
		for _, stmt := range n.Body.List {
			s.bound = copyBound(saved)
			s.ptrAliases = copyPtrMap(savedPtrs)
			s.restoreFlushed(pre)
			cc := stmt.(*ast.CaseClause)
			for _, st := range cc.Body {
				s.walkStmt(st)
			}
		}
		s.bound = saved
		s.ptrAliases = savedPtrs
		s.restoreFlushed(pre)
	case *ast.SelectStmt:
		pre := s.snapshotFlushed()
		savedPtrs := copyPtrMap(s.ptrAliases)
		for _, stmt := range n.Body.List {
			s.ptrAliases = copyPtrMap(savedPtrs)
			s.restoreFlushed(pre)
			cc := stmt.(*ast.CommClause)
			if cc.Comm != nil {
				s.walkStmt(cc.Comm)
			}
			for _, st := range cc.Body {
				s.walkStmt(st)
			}
		}
		s.ptrAliases = savedPtrs
		s.restoreFlushed(pre)
	case *ast.ForStmt:
		if n.Init != nil {
			s.walkStmt(n.Init)
		}
		if n.Cond != nil {
			s.flagReads++
			s.walkExpr(n.Cond)
			s.flagReads--
		}
		pre := s.snapshotFlushed()
		savedPtrs := copyPtrMap(s.ptrAliases)
		s.walkBlock(n.Body)
		if n.Post != nil {
			s.walkStmt(n.Post)
		}
		s.ptrAliases = savedPtrs
		s.restoreFlushed(pre)
	case *ast.RangeStmt:
		s.flagReads++
		s.walkExpr(n.X)
		s.flagReads--
		for _, e := range []ast.Expr{n.Key, n.Value} {
			if id, ok := unwrapParen(e).(*ast.Ident); ok && n.Tok == token.DEFINE {
				s.locals[id.Name] = true
			}
		}
		pre := s.snapshotFlushed()
		savedPtrs := copyPtrMap(s.ptrAliases)
		s.walkBlock(n.Body)
		s.ptrAliases = savedPtrs
		s.restoreFlushed(pre)
	case *ast.GoStmt, *ast.DeferStmt:
		var call *ast.CallExpr
		if g, ok := n.(*ast.GoStmt); ok {
			call = g.Call
		} else {
			call = n.(*ast.DeferStmt).Call
		}
		if lit, ok := unwrapParen(call.Fun).(*ast.FuncLit); ok {
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

func (s *funcScan) walkCond(cond ast.Expr) (map[*binding]bool, map[*binding]bool) {
	switch c := unwrapParen(cond).(type) {
	case *ast.UnaryExpr:
		if c.Op == token.NOT {
			tx, fx := s.walkCond(c.X)
			return fx, tx
		}
	case *ast.BinaryExpr:
		if c.Op == token.LAND {
			tx, fx := s.walkCond(c.X)
			s.restoreFlushed(tx)
			ty, fy := s.walkCond(c.Y)
			return ty, intersectFlushedMaps(fx, fy)
		}
		if c.Op == token.LOR {
			tx, fx := s.walkCond(c.X)
			s.restoreFlushed(fx)
			ty, fy := s.walkCond(c.Y)
			return intersectFlushedMaps(tx, ty), fy
		}
	}
	s.flagReads++
	s.walkExpr(cond)
	s.flagReads--
	snap := s.snapshotFlushed()
	return snap, snap
}

func (s *funcScan) exprsContainFlush(exprs []ast.Expr) bool {
	found := false
	for _, e := range exprs {
		ast.Inspect(e, func(n ast.Node) bool {
			if found {
				return false
			}
			if call, ok := n.(*ast.CallExpr); ok && s.isUIFlushCall(call) {
				found = true
				return false
			}
			return true
		})
	}
	return found
}

func (s *funcScan) isUICall(call *ast.CallExpr) bool {
	name := s.resolvedCallName(call)
	if _, ok := controlBoundIndices(call, name, s.pkg); ok {
		return true
	}
	if s.isUIFlushCall(call) || deferredCallback(name) {
		return true
	}
	switch fun := unwrapParen(call.Fun).(type) {
	case *ast.Ident:
		if fun.Name == "append" || fun.Name == "copy" || fun.Name == "delete" || fun.Name == "clear" || fun.Name == "close" || fun.Name == "panic" || fun.Name == "print" || fun.Name == "println" {
			return true
		}
		if s.pkg != nil && (s.pkg.funcUIBuilder[fun.Name] || s.pkg.funcFlushes[fun.Name] || len(s.pkg.funcBindParams[fun.Name]) > 0) {
			return true
		}
	case *ast.SelectorExpr:
		if fun.Sel.Name == "Key" {
			return true
		}
		if pkg, ok := unwrapParen(fun.X).(*ast.Ident); ok && (knownUIPkgs[pkg.Name] || pureDisplayPkgs[pkg.Name]) {
			return true
		}
		if isUIReceiverExpr(fun.X, s.uiElemVars, s.pkg) {
			return true
		}
		if selRoot := exprKey(fun.X); selRoot != "" && len(s.elemVars[selRoot]) > 0 {
			return true
		}
		if s.pkg != nil && (s.pkg.methodUIBuilder[fun.Sel.Name] || s.pkg.methodFlushes[fun.Sel.Name] || len(s.pkg.methodBindParams[fun.Sel.Name]) > 0) {
			return true
		}
	}
	// Also treat calls whose first arg is the active UI context ident `c` as UI builders
	// unless summarized in the current package as non-builder helper? Wait: checking pkg summary above handles same-package helpers!
	return false
}

func (s *funcScan) isUIFlushCall(n *ast.CallExpr) bool {
	if s.deferDepth > 0 {
		return false
	}
	name := s.resolvedCallName(n)
	if isPkgFuncCall(n) {
		if target, ok := s.fnAliases[callName(n)]; ok && flushCall[target] {
			return true
		}
		return s.pkg != nil && s.pkg.funcFlushes[name]
	}
	if s.pkg != nil && s.pkg.methodFlushes[name] {
		return true
	}
	if flushCall[name] {
		if sel, ok := unwrapParen(n.Fun).(*ast.SelectorExpr); ok {
			if isUIReceiverExpr(sel.X, s.uiElemVars, s.pkg) {
				return true
			}
			if root := exprRootName(exprKey(sel.X)); root != "" && len(s.elemVars[root]) > 0 {
				return true
			}
		}
	}
	return false
}

func (s *funcScan) resolvedCallName(call *ast.CallExpr) string {
	if id, ok := unwrapParen(call.Fun).(*ast.Ident); ok {
		if target, ok := s.fnAliases[id.Name]; ok {
			return target
		}
	}
	return callName(call)
}

func (s *funcScan) walkCallWithDecisionArgs(call *ast.CallExpr) {
	if sel, ok := unwrapParen(call.Fun).(*ast.SelectorExpr); ok {
		s.walkExpr(sel.X)
	} else {
		s.walkExpr(call.Fun)
	}
	name := s.resolvedCallName(call)
	for i, arg := range call.Args {
		s.checkHelperSnapshotArg(name, i, arg, call)
		saved := s.deferArg
		s.deferArg = deferredCallback(name)
		_, isLit := unwrapParen(arg).(*ast.FuncLit)
		if !isLit {
			s.flagReads++
			s.callArgReads++
		}
		s.walkExpr(arg)
		if !isLit {
			s.flagReads--
			s.callArgReads--
		}
		s.deferArg = saved
	}
}

// checkHelperSnapshotArg catches `w.helper(before)` where `before` captured
// `&expr` before an unflushed control and `helper` compares parameter i against
// a field matching `expr` (e.g. `invalidateChangedLargeAcknowledgement(before)`).
func (s *funcScan) checkHelperSnapshotArg(name string, argIdx int, arg ast.Expr, call *ast.CallExpr) {
	if s.pkg == nil || s.flagReads == 0 && s.deferDepth > 0 || s.deferDepth > 0 {
		return
	}
	var fields map[string]bool
	if isPkgFuncCall(call) {
		fields = s.pkg.funcSnapshotFields[name][argIdx]
	} else {
		fields = s.pkg.methodSnapshotField[name][argIdx]
	}
	if len(fields) == 0 {
		return
	}
	id, ok := unwrapParen(arg).(*ast.Ident)
	if !ok {
		return
	}
	snap, ok := s.snaps[id.Name]
	if !ok || snap.root == "" {
		return
	}
	if !fields[leafFieldName(snap.root)] {
		return
	}
	b := s.liveBinding(snap.root)
	if b == nil || b.flushed || b.reported {
		return
	}
	b.reported = true
	b.read = true
	s.fail(BoundViolation{
		File: s.file,
		Line: s.fset.Position(arg.Pos()).Line,
		Msg: "bound value " + strconv.Quote(snap.root) + " checked via helper snapshot " + strconv.Quote(id.Name) +
			" before an input query flushed it: helper sees last frame's value — use the element's Changed() instead",
	})
}

// walkExpr processes one expression in source order: calls record bindings or
// flushes after their receiver and arguments are walked.
func (s *funcScan) walkExpr(n ast.Expr) {
	switch n := n.(type) {
	case *ast.CallExpr:
		var spine []string
		if sel, ok := unwrapParen(n.Fun).(*ast.SelectorExpr); ok {
			s.walkExpr(sel.X)
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
		name := s.resolvedCallName(n)
		for _, nm := range spine {
			s.exempt[nm]++
		}
		_, chainRecv := unwrapParen(n.Fun).(*ast.SelectorExpr)
		chainConfig := false
		if chainRecv {
			_, chainConfig = unwrapParen(unwrapParen(n.Fun).(*ast.SelectorExpr).X).(*ast.CallExpr)
		}
		for i, arg := range n.Args {
			s.checkHelperSnapshotArg(name, i, arg, n)
			saved := s.deferArg
			s.deferArg = deferredCallback(name)
			_, isLit := unwrapParen(arg).(*ast.FuncLit)
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
		if idxs, ok := controlBoundIndices(n, name, s.pkg); ok {
			s.bindArgs(n, idxs)
		} else if s.isUIFlushCall(n) {
			s.flush()
		}
	case *ast.BinaryExpr:
		if isCompareOp(n.Op) {
			s.checkSnapshotCompare(n)
		}
		s.walkExpr(n.X)
		if n.Op == token.LAND || n.Op == token.LOR {
			// Short-circuit RHS only executes conditionally; do not let flushes on
			// the RHS escape past the binary expression.
			pre := s.snapshotFlushed()
			s.walkExpr(n.Y)
			s.restoreFlushed(pre)
		} else {
			s.walkExpr(n.Y)
		}
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			// &x / &cfg.On aliases the bound value rather than reading it.
			if key := exprKey(n.X); key != "" {
				s.markRead(key)
				if id, ok := unwrapParen(n.X).(*ast.Ident); ok {
					s.markRead(id.Name)
				}
			}
			if sel, ok := unwrapParen(n.X).(*ast.SelectorExpr); ok {
				if root := exprRootName(exprKey(sel)); root == "" || !s.valueLocals[root] {
					s.walkExpr(sel.X)
				}
			}
			return
		}
		s.walkExpr(n.X)
	case *ast.Ident:
		if s.locals[n.Name] {
			s.readLocal(n.Name, n.Pos())
		}
	case *ast.SelectorExpr:
		if key := exprKey(n); key != "" && s.valueLocals[exprRootName(key)] {
			if bs := s.bound[key]; bs != nil && len(bs.local) > 0 {
				s.readLocal(key, n.Pos())
				return
			}
		}
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
		// *p dereferences a bound pointer: flag when passed into a non-UI
		// statement call (e.g. Store.SetDraft(..., *m.Draft), onAcknowledge(*ack))
		// before flush, or when underlying local storage is stack-owned.
		// Display-only UI render/filter reads (`if *checked`, `strings.TrimSpace(*m.Draft)`)
		// stay legal unless compared against a before-snapshot.
		if key := exprKey(n.X); key != "" {
			if b := s.livePtr(key); b != nil {
				if s.callArgReads > 0 && s.deferDepth == 0 && s.exempt[key] == 0 && s.exemptAll == 0 {
					s.flagRead(b, n.Pos())
				}
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
	savedLocals := s.locals
	savedValLocals := s.valueLocals
	savedFnAliases := s.fnAliases
	savedPtrAliases := s.ptrAliases
	savedUIVars := s.uiElemVars
	s.locals = copyBoolMap(savedLocals)
	s.valueLocals = copyBoolMap(savedValLocals)
	s.fnAliases = copyStringMap(savedFnAliases)
	s.ptrAliases = copyPtrMap(savedPtrAliases)
	s.uiElemVars = copyBoolMap(savedUIVars)
	if n.Type.Params != nil {
		for _, field := range n.Type.Params.List {
			valType := isValueStructType(field.Type)
			for _, name := range field.Names {
				s.locals[name.Name] = true
				if valType {
					s.valueLocals[name.Name] = true
				}
			}
		}
	}
	s.walkBlock(n.Body)
	s.locals = savedLocals
	s.valueLocals = savedValLocals
	s.fnAliases = savedFnAliases
	s.ptrAliases = savedPtrAliases
	s.uiElemVars = savedUIVars
}

func (s *funcScan) isReadOnlyHelperArg(call *ast.CallExpr, name string, argIdx int) bool {
	roMap := summaryReadOnlyMap(call, name, s.pkg)
	if len(roMap) == 0 {
		return false
	}
	roIdx, ok := roMap[argIdx]
	return ok && roIdx >= 0 && roIdx < len(call.Args) && isLiteralTrueExpr(call.Args[roIdx])
}

// bindArgs records deferred writebacks for a control call's bound-pointer
// argument positions: `&expr` on stack-owned storage (`x`, `cfg.On`), `&s.F`
// on persistent fields, pointer aliases (`p := &v`), and bare pointers (`m.Draft`).
func (s *funcScan) bindArgs(call *ast.CallExpr, indices []int) {
	name := s.resolvedCallName(call)
	for _, idx := range indices {
		if idx < 0 || idx >= len(call.Args) {
			continue
		}
		if s.isReadOnlyHelperArg(call, name, idx) {
			continue
		}
		rawArg := unwrapParen(call.Args[idx])
		if u, ok := rawArg.(*ast.UnaryExpr); ok && u.Op == token.AND {
			if s.readOnly[u.Pos()] {
				continue
			}
			key := exprKey(u.X)
			if key == "" {
				continue
			}
			s.bind(key, s.isStackOwnedExpr(u.X, key), u.Pos())
			continue
		}
		if s.readOnly[rawArg.Pos()] {
			continue
		}
		if id, ok := rawArg.(*ast.Ident); ok && id.Name == "nil" {
			continue
		}
		// Handle `*pp` dereferenced pointer expressions and plain bare pointers (`p`, `m.Draft`).
		targetExpr := rawArg
		if star, ok := rawArg.(*ast.StarExpr); ok {
			targetExpr = star.X
		}
		key := exprKey(targetExpr)
		if key == "" {
			continue
		}
		if pt, ok := s.ptrAliases[key]; ok {
			s.bind(pt.key, pt.local, rawArg.Pos())
			s.bind(key, false, rawArg.Pos())
			continue
		}
		s.bind(key, false, rawArg.Pos())
	}
}

// markReadOnlyArgs precomputes the bound arg positions whose control sits under
// a ReadOnly(true) chain: a read-only control never accepts input, so its
// bound local is display-only and exempt.
func (s *funcScan) markReadOnlyArgs(body *ast.BlockStmt) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || callName(call) != "ReadOnly" || len(call.Args) != 1 {
			return true
		}
		if id, ok := unwrapParen(call.Args[0]).(*ast.Ident); !ok || id.Name != "true" {
			return true
		}
		sel, ok := unwrapParen(call.Fun).(*ast.SelectorExpr)
		if !ok {
			return true
		}
		for _, c := range spineCalls(sel.X) {
			idxs, ok := controlBoundIndices(c, s.resolvedCallName(c), s.pkg)
			if !ok {
				continue
			}
			for _, idx := range idxs {
				if idx >= 0 && idx < len(c.Args) {
					s.readOnly[unwrapParen(c.Args[idx]).Pos()] = true
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
		call, ok := unwrapParen(e).(*ast.CallExpr)
		if !ok {
			return out
		}
		out = append(out, call)
		sel, ok := unwrapParen(call.Fun).(*ast.SelectorExpr)
		if !ok {
			return out
		}
		e = sel.X
	}
}

// spineLocalBinds returns the bound value keys owned by deferred controls
// along the receiver chain under e — reads of them inside the control's own
// builder/render arguments still run during the control's construction chain.
func (s *funcScan) spineLocalBinds(e ast.Expr) []string {
	var out []string
	for _, c := range spineCalls(e) {
		cname := s.resolvedCallName(c)
		idxs, ok := controlBoundIndices(c, cname, s.pkg)
		if !ok {
			continue
		}
		for _, idx := range idxs {
			if idx < 0 || idx >= len(c.Args) {
				continue
			}
			if s.isReadOnlyHelperArg(c, cname, idx) {
				continue
			}
			raw := unwrapParen(c.Args[idx])
			if u, ok := raw.(*ast.UnaryExpr); ok && u.Op == token.AND {
				if key := exprKey(u.X); key != "" && s.isStackOwnedExpr(u.X, key) {
					out = append(out, key)
				}
				continue
			}
			target := raw
			if star, ok := raw.(*ast.StarExpr); ok {
				target = star.X
			}
			if key := exprKey(target); key != "" {
				if pt, ok := s.ptrAliases[key]; ok && pt.local {
					out = append(out, pt.key)
				}
				out = append(out, key)
			}
		}
	}
	return out
}

// checkSnapshotCompare implements pattern (b): `before := expr` captured
// before a deferred control bound &expr, then `expr != before` (or ==, <, <=,
// >, >=, either operand order, including parenthesized operands or pure-wrapper
// snapshots) compared before any flushing UI query.
func (s *funcScan) checkSnapshotCompare(n *ast.BinaryExpr) {
	for _, pair := range [2][2]ast.Expr{{n.X, n.Y}, {n.Y, n.X}} {
		id, ok := unwrapParen(pair[0]).(*ast.Ident)
		if !ok {
			continue
		}
		snap, ok := s.snaps[id.Name]
		if !ok || snap.root == "" {
			continue
		}
		rhsKey := exprKey(pair[1])
		rhsCanon := exprCanon(pair[1])
		if rhsKey != snap.root && (rhsCanon == "" || rhsCanon != snap.canon) {
			continue
		}
		b := s.liveBinding(snap.root)
		if b == nil || b.flushed || b.reported || s.flagReads == 0 || s.deferDepth > 0 {
			continue
		}
		b.reported = true
		b.read = true
		s.fail(BoundViolation{
			File: s.file,
			Line: s.fset.Position(n.Pos()).Line,
			Msg: "bound value " + strconv.Quote(snap.root) + " compared against snapshot " + strconv.Quote(id.Name) +
				" before an input query flushed it: compare sees last frame's value — use the element's Changed() instead",
		})
	}
}

// callName resolves a call's simple name: ui.Checkbox(...) -> "Checkbox",
// f(...) -> "f".
func callName(c *ast.CallExpr) string {
	switch f := unwrapParen(c.Fun).(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

func copyBoolMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyPtrMap(in map[string]ptrTarget) map[string]ptrTarget {
	out := make(map[string]ptrTarget, len(in))
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
