package diffsurface

import (
	"strings"
	"sync"

	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/widgets/diff"
)

type patchItemKind uint8

const (
	patchItemFileHeader patchItemKind = iota
	patchItemSplitHeader
	patchItemHunkHeader
	patchItemMetaLine
	patchItemUnifiedLine
	patchItemSplitRow
)

type patchRowKey struct {
	file  string
	kind  patchItemKind
	hunk  int
	row   int
	split bool
	wrap  bool
}

type patchListItem struct {
	key       patchRowKey
	fileIndex int
	file      patchFile
	hunk      int
	line      diffLine
	splitRow  patchRow
	fileMinW  float32
}

// diffToolButton paints one compact control of the patch tool row.
func diffToolButton(c *ui.Context, label string, selected bool, onClick func()) ui.Element {
	button := ui.Button(c, label).Height(26).Padding(0, 10).Radius(6).BorderWidth(0).FontSize(fonttheme.InterfaceSize(12)).Selected(selected).Tooltip(label)
	if selected {
		button.Background(c.Theme().SurfaceHover).TextColor(c.Theme().Text)
	} else {
		button.Background(ui.Transparent).TextColor(c.Theme().TextMuted)
	}
	if button.Hovered() && !selected {
		button.Background(ui.RGBA(c.Theme().TextMuted.R, c.Theme().TextMuted.G, c.Theme().TextMuted.B, .14))
	}
	if button.Clicked() && onClick != nil {
		onClick()
	}
	return button
}

// paintDiffControls paints the tool row above the patch: the view mode as a
// two-chip choice, then wrap and fold controls.
func paintDiffControls(c *ui.Context, m Model) {
	if m.Workbench == nil {
		return
	}
	view := &m.Workbench.View
	source := ""
	if m.Text != nil {
		source = *m.Text
	}
	view.update(source)
	ui.Row(c).FillWidth().Gap(4).Padding(6, 12, 6, 12).Wrap().AlignItems(ui.Center).Children(func() {
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffUnified), !view.Split, func() {
			view.Split = false
			view.bumpLayout()
		})
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffSplit), view.Split, func() {
			view.Split = true
			view.bumpLayout()
		})
		ui.Box(c).Width(0.5).Height(16).Shrink(0).Background(c.Theme().Border)
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffWrap), view.Wrap, func() {
			view.Wrap = !view.Wrap
			view.bumpLayout()
		})
		ui.Box(c).Grow(1)
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffExpandAll), false, func() { view.setAllClosed(false); view.setAllHunks(false) })
		diffToolButton(c, i18n.T(i18n.KeySurfacesDiffCollapseAll), false, func() { view.setAllClosed(true); view.setAllHunks(true) })
	})
}

type preparedPatchDoc struct {
	source  string
	files   []patchFile
	split   bool
	wrap    bool
	items   []patchListItem
	maxRowW float32
}

const maxPrebuiltPatchCache = 8

var prebuiltPatchCache struct {
	mu    sync.Mutex
	order []string
	docs  map[string]*preparedPatchDoc
}

func clearPrebuiltPatchCache() {
	prebuiltPatchCache.mu.Lock()
	clear(prebuiltPatchCache.docs)
	prebuiltPatchCache.order = nil
	prebuiltPatchCache.mu.Unlock()
}

func preparedCacheKey(source string, split, wrap bool) string {
	mode := byte('u')
	if split {
		mode = 's'
	}
	w := byte('n')
	if wrap {
		w = 'w'
	}
	return string([]byte{mode, w, ':'}) + source
}

func storePrebuiltPatchDoc(doc *preparedPatchDoc) {
	if doc == nil || strings.TrimSpace(doc.source) == "" {
		return
	}
	key := preparedCacheKey(doc.source, doc.split, doc.wrap)
	prebuiltPatchCache.mu.Lock()
	if prebuiltPatchCache.docs == nil {
		prebuiltPatchCache.docs = make(map[string]*preparedPatchDoc, maxPrebuiltPatchCache)
	}
	if _, exists := prebuiltPatchCache.docs[key]; !exists {
		prebuiltPatchCache.order = append(prebuiltPatchCache.order, key)
		for len(prebuiltPatchCache.order) > maxPrebuiltPatchCache {
			evict := prebuiltPatchCache.order[0]
			prebuiltPatchCache.order = prebuiltPatchCache.order[1:]
			delete(prebuiltPatchCache.docs, evict)
		}
	}
	prebuiltPatchCache.docs[key] = doc
	prebuiltPatchCache.mu.Unlock()
}

func lookupPrebuiltPatchDoc(source string, split, wrap bool) *preparedPatchDoc {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	key := preparedCacheKey(source, split, wrap)
	prebuiltPatchCache.mu.Lock()
	doc := prebuiltPatchCache.docs[key]
	prebuiltPatchCache.mu.Unlock()
	return doc
}

func lookupPrebuiltPatchFiles(source string) *preparedPatchDoc {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	prebuiltPatchCache.mu.Lock()
	for _, doc := range prebuiltPatchCache.docs {
		if doc.source == source {
			prebuiltPatchCache.mu.Unlock()
			return doc
		}
	}
	prebuiltPatchCache.mu.Unlock()
	return nil
}

func measurePatchFilesMetrics(files []patchFile, counter *int) {
	for i := range files {
		f := &files[i]
		if f.metricsReady {
			continue
		}
		if counter != nil {
			*counter++
		}
		f.splitColW = splitColumnWidth(f.Rows)
		f.uniLines = unifiedLines(f.Rows)
		f.uniMinW = unifiedMinWidth(f.Rows) + diffNumberWidth + diffSignWidth + 28
		f.metricsReady = true
	}
}

func prebuildPatchDocument(source string, split, wrap bool) *preparedPatchDoc {
	if strings.TrimSpace(source) == "" {
		return nil
	}
	if existing := lookupPrebuiltPatchDoc(source, split, wrap); existing != nil {
		return existing
	}
	var files []patchFile
	if base := lookupPrebuiltPatchFiles(source); base != nil {
		files = base.files
	} else {
		files = parsePatchFiles(source)
		measurePatchFilesMetrics(files, nil)
	}
	items, maxRowW := buildPatchItemsFromFiles(files, split, wrap, nil, nil, nil)
	doc := &preparedPatchDoc{
		source:  source,
		files:   files,
		split:   split,
		wrap:    wrap,
		items:   items,
		maxRowW: maxRowW,
	}
	storePrebuiltPatchDoc(doc)
	return doc
}

func buildPatchListItems(view *patchViewState, source string) ([]patchListItem, float32) {
	files := view.update(source)
	sig := view.layoutSignature()
	if view.cachedItems != nil &&
		view.cachedSrc == view.source &&
		view.cachedSplit == view.Split &&
		view.cachedWrap == view.Wrap &&
		view.cachedSeq == view.layoutSeq &&
		view.cachedSig == sig {
		return view.cachedItems, view.cachedMaxW
	}
	if !hasAnyClosedFile(view.Closed) && !hasAnyFoldedHunk(view.Folded) {
		if pre := lookupPrebuiltPatchDoc(view.source, view.Split, view.Wrap); pre != nil && pre.items != nil {
			view.cachedItems = pre.items
			view.cachedMaxW = pre.maxRowW
			view.cachedSrc = view.source
			view.cachedSplit = view.Split
			view.cachedWrap = view.Wrap
			view.cachedSeq = view.layoutSeq
			view.cachedSig = sig
			return view.cachedItems, view.cachedMaxW
		}
	}

	view.uiBuildCalls++
	items, maxRowW := buildPatchItemsFromFiles(files, view.Split, view.Wrap, view.Closed, view.Folded, &view.uiMeasureCalls)

	view.cachedItems = items
	view.cachedMaxW = maxRowW
	view.cachedSrc = view.source
	view.cachedSplit = view.Split
	view.cachedWrap = view.Wrap
	view.cachedSeq = view.layoutSeq
	view.cachedSig = sig
	return items, maxRowW
}

func buildPatchItemsFromFiles(files []patchFile, split, wrap bool, closed map[string]bool, foldedMap map[string]map[int]bool, measureCounter *int) ([]patchListItem, float32) {
	items := make([]patchListItem, 0, len(files)*8)
	var maxRowW float32
	for fileIndex, f := range files {
		if !f.metricsReady {
			if measureCounter != nil {
				*measureCounter++
			}
			f.splitColW = splitColumnWidth(f.Rows)
			f.uniLines = unifiedLines(f.Rows)
			f.uniMinW = unifiedMinWidth(f.Rows) + diffNumberWidth + diffSignWidth + 28
			f.metricsReady = true
		}
		items = append(items, patchListItem{
			key:       patchRowKey{file: f.Path, kind: patchItemFileHeader, hunk: -1, row: fileIndex, split: split, wrap: wrap},
			fileIndex: fileIndex,
			file:      f,
			hunk:      -1,
		})
		if closed[f.Path] {
			continue
		}
		folded := foldedMap[f.Path]
		if split {
			colMinW := float32(0)
			if !wrap {
				colMinW = f.splitColW
				if total := colMinW*2 + 8; total > maxRowW {
					maxRowW = total
				}
			}
			items = append(items, patchListItem{
				key:       patchRowKey{file: f.Path, kind: patchItemSplitHeader, hunk: -1, row: -1, split: split, wrap: wrap},
				fileIndex: fileIndex,
				file:      f,
				hunk:      -1,
				fileMinW:  colMinW,
			})
			hunk, hidden := -1, false
			for rowIndex, r := range f.Rows {
				if r.Meta != "" {
					if len(r.Meta) > 1 && r.Meta[0] == '@' && r.Meta[1] == '@' {
						hunk++
						hidden = folded[hunk]
						items = append(items, patchListItem{
							key:       patchRowKey{file: f.Path, kind: patchItemHunkHeader, hunk: hunk, row: rowIndex, split: split, wrap: wrap},
							fileIndex: fileIndex,
							file:      f,
							hunk:      hunk,
							line:      diffLine{Kind: diffLineHunk, Text: r.Meta, Hunk: hunk},
							fileMinW:  colMinW,
						})
						continue
					}
					items = append(items, patchListItem{
						key:       patchRowKey{file: f.Path, kind: patchItemMetaLine, hunk: hunk, row: rowIndex, split: split, wrap: wrap},
						fileIndex: fileIndex,
						file:      f,
						hunk:      hunk,
						line:      diffLine{Kind: diffLineMeta, Text: r.Meta},
						fileMinW:  colMinW,
					})
					continue
				}
				if hidden {
					continue
				}
				items = append(items, patchListItem{
					key:       patchRowKey{file: f.Path, kind: patchItemSplitRow, hunk: hunk, row: rowIndex, split: split, wrap: wrap},
					fileIndex: fileIndex,
					file:      f,
					hunk:      hunk,
					splitRow:  r,
					fileMinW:  colMinW,
				})
			}
			continue
		}

		uniMinW := float32(0)
		if !wrap {
			uniMinW = f.uniMinW
			if uniMinW > maxRowW {
				maxRowW = uniMinW
			}
		}
		allLines := f.uniLines
		hunk, hidden := -1, false
		for lineIndex, line := range allLines {
			if line.Kind == diffLineHunk {
				hunk++
				hidden = folded[hunk]
				line.Hunk = hunk
				items = append(items, patchListItem{
					key:       patchRowKey{file: f.Path, kind: patchItemHunkHeader, hunk: hunk, row: lineIndex, split: split, wrap: wrap},
					fileIndex: fileIndex,
					file:      f,
					hunk:      hunk,
					line:      line,
					fileMinW:  uniMinW,
				})
				continue
			}
			if hidden {
				continue
			}
			kind := patchItemUnifiedLine
			if line.Kind == diffLineMeta {
				kind = patchItemMetaLine
			}
			items = append(items, patchListItem{
				key:       patchRowKey{file: f.Path, kind: kind, hunk: hunk, row: lineIndex, split: split, wrap: wrap},
				fileIndex: fileIndex,
				file:      f,
				hunk:      hunk,
				line:      line,
				fileMinW:  uniMinW,
			})
		}
	}
	return items, maxRowW
}

// paintPatchFiles paints every file of a patch using ui.List row virtualization.
func paintPatchFiles(c *ui.Context, m Model, source string, st diff.Style) {
	if m.Workbench == nil {
		host := ui.Column(c).Fill().MinHeight(0)
		view := ui.Local(host, "diffsurface-standalone-patch-view", func() patchViewState { return patchViewState{} })
		host.Children(func() {
			paintVirtualizedPatchList(c, m, view, source, st)
		})
		return
	}
	paintVirtualizedPatchList(c, m, &m.Workbench.View, source, st)
}

func paintVirtualizedPatchList(c *ui.Context, m Model, view *patchViewState, source string, st diff.Style) {
	items, maxRowW := buildPatchListItems(view, source)
	if len(items) == 0 {
		return
	}
	// Copy reads the same logical rows as the painter, including rows that
	// virtualization has never built. layoutSeq changes on source replacement
	// and fold/view changes; row keys keep the remaining endpoints attached.
	textRows := func(i int) []string { return patchSelectableParagraphs(items[i]) }
	if view.Wrap {
		ui.List(c, view.ensureList(), len(items)).
			ItemKey(func(i int) any { return items[i].key }).
			SelectionText(view.layoutSeq, textRows).
			Fill().
			MinHeight(0).
			Gap(1).
			Padding(8, 8, 8, 8).
			Selectable().
			Rows(func(row ui.ListRow) {
				paintPatchListItem(row.Context, m, view, items[row.Index], st, 0)
			})
		return
	}
	listW := max(view.viewportW, maxRowW+16)
	rowMinW := max(listW-16, 0)
	inv := c.Services().Invalidate
	ui.ScrollHorizontal(c).Fill().MinHeight(0).ObserveLayout(func(box ui.Rect) {
		if box.W > 0 && box.W != view.viewportW {
			view.viewportW = box.W
			if inv != nil {
				inv()
			}
		}
	}).Children(func() {
		ui.List(c, view.ensureList(), len(items)).
			ItemKey(func(i int) any { return items[i].key }).
			SelectionText(view.layoutSeq, textRows).
			Width(listW).
			FillHeight().
			MinHeight(0).
			Gap(1).
			Padding(8, 8, 8, 8).
			Selectable().
			Rows(func(row ui.ListRow) {
				paintPatchListItem(row.Context, m, view, items[row.Index], st, rowMinW)
			})
	})
}

// patchSelectableParagraphs follows paintPatchListItem's selectable build order.
// Headers and gutters are explicitly unselectable. A split row has two logical
// paragraphs; endpoint clipping must not append the cell after the selected one.
func patchSelectableParagraphs(item patchListItem) []string {
	switch item.key.kind {
	case patchItemMetaLine, patchItemUnifiedLine:
		return []string{item.line.Text}
	case patchItemSplitRow:
		return []string{splitCellText(item.splitRow.Old), splitCellText(item.splitRow.New)}
	default:
		return nil
	}
}

func splitCellText(line string) string {
	if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
		return line[1:]
	}
	return line
}

func paintRecordedPatchFiles(c *ui.Context, source string, split, wrap bool, st diff.Style) {
	w := &Workbench{}
	w.View.Split, w.View.Wrap = split, wrap
	items, maxRowW := buildPatchListItems(&w.View, source)
	if len(items) == 0 {
		return
	}
	m := Model{Workbench: w}
	if wrap {
		ui.Column(c).FillWidth().MinWidth(0).Gap(1).Padding(8, 8, 8, 8).Selectable().Children(func() {
			for _, item := range items {
				paintPatchListItem(c, m, &w.View, item, st, 0)
			}
		})
		return
	}
	ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
		ui.Column(c).MinWidth(maxRowW+16).Gap(1).Padding(8, 8, 8, 8).Selectable().Children(func() {
			for _, item := range items {
				paintPatchListItem(c, m, &w.View, item, st, maxRowW)
			}
		})
	})
}

func paintPatchListItem(c *ui.Context, m Model, view *patchViewState, item patchListItem, st diff.Style, listRowMinW float32) {
	switch item.key.kind {
	case patchItemFileHeader:
		f := item.file
		open := !view.Closed[f.Path]
		topPad := float32(0)
		if item.fileIndex > 0 {
			topPad = 6
		}
		header := ui.Row(c).MinWidth(0).Gap(6).Padding(topPad, 0, 2, 0).AlignItems(ui.Center).Unselectable()
		if !view.Wrap && view.viewportW > 16 {
			header.Width(view.viewportW - 16)
		} else {
			header.FillWidth()
		}
		header.Children(func() {
			title := f.Path
			if title == "" {
				title = word("差异内容", "Diff contents")
			}
			head := ui.Button(c, title).FontSize(fonttheme.InterfaceSize(12)).Grow(1).MinWidth(0).Height(28).Radius(6).Padding(0, 8).BorderWidth(0).Background(ui.Transparent).TextColor(st.FG).Disabled(m.OnOpenFile == nil || !safePatchPath(f.Path)).Tooltip(f.Path)
			if head.Hovered() && m.OnOpenFile != nil && safePatchPath(f.Path) {
				head.Background(ui.RGBA(st.Muted.R, st.Muted.G, st.Muted.B, .14))
			}
			if head.Clicked() && m.OnOpenFile != nil {
				if err := m.OnOpenFile(f.Path); err != nil && m.Workbench != nil {
					m.Workbench.Status = err.Error()
				}
			}
			paintPatchFileActions(c, m, f.Path)
			fold := i18n.T(i18n.KeySurfacesDiffExpandAll)
			if open {
				fold = i18n.T(i18n.KeySurfacesDiffCollapseAll)
			}
			toggle := ui.Button(c, "").Label(fold).Size(22, 22).Shrink(0).Padding(0).Radius(4).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Tooltip(fold)
			toggle.Children(func() {
				if open {
					ui.Icon(c, diffChevronDownIcon).Size(10, 10).TextColor(st.Muted).PassThrough()
				} else {
					ui.Icon(c, diffChevronRightIcon).Size(10, 10).TextColor(st.Muted).PassThrough()
				}
			})
			if toggle.Hovered() {
				toggle.Background(ui.RGBA(st.Muted.R, st.Muted.G, st.Muted.B, .14))
			}
			if toggle.Clicked() {
				if view.Closed == nil {
					view.Closed = map[string]bool{}
				}
				view.Closed[f.Path] = open
				view.bumpLayout()
			}
		})

	case patchItemSplitHeader:
		minWidth := float32(0)
		if !view.Wrap {
			minWidth = item.fileMinW
		}
		ui.Row(c).FillWidth().Gap(8).Padding(0, 8, 4, 8).Unselectable().Children(func() {
			ui.Text(c, i18n.T(i18n.KeySurfacesDiffBefore)).Grow(1).MinWidth(minWidth).FontSize(fonttheme.InterfaceSize(11)).TextColor(st.Muted)
			ui.Text(c, i18n.T(i18n.KeySurfacesDiffAfter)).Grow(1).MinWidth(minWidth).FontSize(fonttheme.InterfaceSize(11)).TextColor(st.Muted)
		})

	case patchItemHunkHeader:
		fPath := item.file.Path
		fold := hunkFold{
			collapsed: view.foldedHunks(fPath),
			onToggle:  func(index int) { view.toggleHunk(fPath, index) },
		}
		ui.Column(c).FillWidth().MinWidth(0).Unselectable().Children(func() {
			paintHunkHeader(c, item.line.Text, item.hunk, st, fold)
		})

	case patchItemMetaLine:
		minWidth := float32(0)
		if !view.Wrap {
			minWidth = item.fileMinW
		}
		paintDiffLine(c, item.line, st, minWidth, view.Wrap)

	case patchItemUnifiedLine:
		minWidth := float32(0)
		if !view.Wrap {
			minWidth = max(listRowMinW, item.fileMinW)
		}
		paintDiffLine(c, item.line, st, minWidth, view.Wrap)

	case patchItemSplitRow:
		minWidth := float32(0)
		if !view.Wrap {
			minWidth = item.fileMinW
		}
		r := item.splitRow
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			paintSplitCell(c, r.Old, r.OldNumber, view.Wrap, minWidth, st)
			paintSplitCell(c, r.New, r.NewNumber, view.Wrap, minWidth, st)
		})
	}
}

// Staging belongs to the file being reviewed, not a context-only action on a
// separate commit checklist. The commit dialog keeps its own explicit selection.
func paintPatchFileActions(c *ui.Context, m Model, path string) {
	w := m.Workbench
	if w == nil || w.BaseRef != "" || !safePatchPath(path) {
		return
	}
	busy := w.Operation != nil || w.isStagingBusy() || w.isDiscardPreparing()
	for _, change := range w.Changes {
		if change.Path != path {
			continue
		}
		if change.Unstaged() && ui.Button(c, word("暂存", "Stage")).Height(24).Shrink(0).Padding(0, 6).FontSize(fonttheme.InterfaceSize(12)).Disabled(busy).Clicked() {
			stageCommitFile(m, change)
		}
		if change.Staged() && ui.Button(c, word("取消暂存", "Unstage")).Height(24).Shrink(0).Padding(0, 6).FontSize(fonttheme.InterfaceSize(12)).Disabled(busy).Clicked() {
			unstageCommitFile(m, change)
		}
		if change.Unstaged() {
			ui.MenuButton(c, word("更多", "More"), func(menu *ui.Menu) {
				if menu.Item(word("还原…", "Discard file…")).Disabled(busy).Chosen() {
					reviewDiscard(m, path)
				}
			}).Height(24).Shrink(0).FontSize(fonttheme.InterfaceSize(12))
		}
		return
	}
}

// splitColumnWidth estimates the pixel width of the widest line in a hunk.
func splitColumnWidth(rows []patchRow) float32 {
	longest := 20
	for _, r := range rows {
		for _, line := range []string{r.Old, r.New} {
			if n := len([]rune(line)); n > longest {
				longest = n
			}
		}
	}
	return float32(longest+7) * 7.5
}
