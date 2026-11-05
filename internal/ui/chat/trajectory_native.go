package chat

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"sort"
	"strings"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/diff"
	"whaleisle/internal/ui/widgets/markdown"
)

type trajectoryFoldState struct {
	turns, calls             map[string]bool
	globalTurns, globalCalls bool
	ready                    bool
}

func (s *trajectoryFoldState) rows(all []trajRow, query string, turns, calls bool) []trajRow {
	if !s.ready || s.globalTurns != turns {
		s.turns = map[string]bool{}
		s.globalTurns = turns
	}
	if !s.ready || s.globalCalls != calls {
		s.calls = map[string]bool{}
		s.globalCalls = calls
	}
	s.ready = true
	var out []trajRow
	turn := ""
	hidden := false
	for i := 0; i < len(all); {
		row := all[i]
		key := row.TurnKey
		// Legacy callers may supply rows before the ledger grouping pass.
		if key == "" && row.Role == "user" && row.Kind == session.MessageKindText {
			key = row.ID
		}
		if key != "" && key != turn {
			turn = key
			fold, ok := s.turns[turn]
			if !ok {
				fold = turns
			}
			hidden = fold
			row.FoldKey = "turn:" + turn
			row.Folded = fold
			out = append(out, row)
			i++
			continue
		}
		if hidden {
			i++
			continue
		}
		if row.Role == "assistant" || row.Kind == session.MessageKindTool || row.Kind == session.MessageKindThinking {
			j := i + 1
			for j < len(all) && (all[j].Role == "assistant" || all[j].Kind == session.MessageKindTool || all[j].Kind == session.MessageKindThinking) && sameTrajectoryStep(all[i].Source, all[j].Source) {
				j++
			}
			key := row.ID
			fold, ok := s.calls[key]
			if !ok {
				fold = calls
			}
			group := row
			group.ID = key
			group.FoldKey = "call:" + key
			group.Folded = fold
			if fold {
				group.Role = "assistant-group"
				group.Kind = session.MessageKindText
				group.Output = trajectoryText("助手步骤", "Assistant step") + fmt.Sprintf(" · %d records", j-i)
				var content trajectoryContent
				for _, part := range all[i:j] {
					content.Input += part.Full.Input + "\n"
					content.Output += part.Full.Output + "\n"
					content.Think += part.Full.Think + "\n"
					content.Tools += part.Full.Tools + "\n"
					content.Raw += part.Full.Raw + "\n"
				}
				group.Full = content
			}
			if j-i > 1 || fold {
				out = append(out, group)
				if !fold {
					out = append(out, all[i+1:j]...)
				}
			} else {
				row.FoldKey = "call:" + key
				row.Folded = false
				out = append(out, row)
			}
			i = j
			continue
		}
		out = append(out, row)
		i++
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return out
	}
	// Searching is a full-record operation, independent of visual folding.
	out = out[:0]
	for _, r := range all {
		hay := strings.ToLower(r.Full.Input + " " + r.Full.Output + " " + r.Full.Think + " " + r.Full.Tools + " " + r.Full.Raw + " " + r.Model)
		if strings.Contains(hay, q) {
			out = append(out, r)
		}
	}
	return out
}
func sameTrajectoryStep(a, b session.Message) bool {
	ta, sa, ka := trajectoryCoordinates(a)
	tb, sb, kb := trajectoryCoordinates(b)
	return !ka && !kb || ka && kb && ta == tb && sa == sb
}
func (s *trajectoryFoldState) toggle(row trajRow) {
	key := strings.TrimPrefix(row.FoldKey, "turn:")
	if strings.HasPrefix(row.FoldKey, "turn:") {
		s.turns[key] = !row.Folded
		return
	}
	key = strings.TrimPrefix(row.FoldKey, "call:")
	s.calls[key] = !row.Folded
}

// A record link is a semantic navigation, not just an inspector selection.
func (s *trajectoryFoldState) reveal(all []trajRow, id string) {
	target := -1
	for i, row := range all {
		if row.ID == id {
			target = i
			break
		}
	}
	if target < 0 {
		return
	}
	row := all[target]
	if row.TurnKey != "" {
		s.turns[row.TurnKey] = false
	}
	for i := target; i >= 0; i-- {
		if sameTrajectoryStep(all[i].Source, row.Source) && (all[i].Role == "assistant" || all[i].Kind == session.MessageKindTool || all[i].Kind == session.MessageKindThinking) {
			s.calls[all[i].ID] = false
		}
		if all[i].TurnKey != row.TurnKey {
			break
		}
	}
}

func trajectorySource(row trajRow) string {
	if len(row.Source.Raw) > 0 {
		return string(row.Source.Raw)
	}
	raw, _ := json.MarshalIndent(row.Source, "", "  ")
	return string(raw)
}
func trajectoryLinks(c *ui.Context, m TrajectoryModel, row trajRow, all []trajRow, selected *string) {
	source := row.Source
	callID, parentID := "", ""
	if source.Tool != nil {
		callID, parentID = source.Tool.CallID, source.Tool.ParentCallID
	}
	var wire struct {
		Data struct {
			CallID, ParentCallID string
			Call                 struct{ ID, ParentID string }
		}
	}
	_ = json.Unmarshal(source.Raw, &wire)
	if callID == "" {
		callID = wire.Data.CallID
	}
	if parentID == "" {
		parentID = wire.Data.ParentCallID
	}
	if callID == "" {
		part := session.ProjectToolMessages([]session.Message{source})
		if len(part) > 0 && part[0].Tool != nil {
			callID = part[0].Tool.CallID
			parentID = part[0].Tool.ParentCallID
		}
	}
	turn, step, known := transcriptEventCoordinates(source)
	targets := map[string]string{}
	for _, candidate := range all {
		if candidate.ID == row.ID {
			continue
		}
		msg := candidate.Source
		if msg.Tool != nil {
			if parentID != "" && msg.Tool.CallID == parentID {
				targets["Parent tool"] = candidate.ID
			}
			if callID != "" && msg.Tool.CallID == callID && msg.EventType == "tool/result" {
				targets["Result"] = candidate.ID
			}
		}
		if callID != "" {
			nodes := session.ProjectToolMessages([]session.Message{msg})
			if len(nodes) > 0 && nodes[0].Tool != nil {
				node := nodes[0].Tool
				if parentID != "" && node.CallID == parentID {
					targets["Parent tool"] = candidate.ID
				}
				if node.CallID == callID && msg.EventType == "tool/result" {
					targets["Result"] = candidate.ID
				}
			}
		}
		t, s, k := transcriptEventCoordinates(msg)
		if known && k && t == turn && s == step && msg.Role == "assistant" && msg.Kind == session.MessageKindText {
			targets["Assistant"] = candidate.ID
		}
	}
	for _, request := range m.Snapshot.TrajectoryRequests {
		if "request:"+request.ID == row.ID && request.ResultSeq != nil {
			for _, candidate := range all {
				if candidate.Source.Seq == *request.ResultSeq {
					targets["Result"] = candidate.ID
				}
			}
		}
	}
	if len(targets) == 0 {
		return
	}
	ui.Row(c).FillWidth().Wrap().Gap(6).Children(func() {
		for _, label := range []string{"Assistant", "Parent tool", "Result"} {
			if id := targets[label]; id != "" {
				if ui.Button(c, label).Label("Trajectory jump " + label).Clicked() {
					*selected = id
				}
			}
		}
	})
}

func trajectoryHistoryPair(snapshot session.ViewState, row trajRow) (before, after string) {
	if row.Source.Kind == session.MessageKindSystemPrompt {
		after = row.Source.Body
		for _, msg := range trajectoryHistoryMessages(snapshot) {
			if msg.ID == row.Source.ID {
				break
			}
			if msg.Kind == session.MessageKindSystemPrompt {
				before = msg.Body
			}
		}
		return
	}
	headers := append([]session.TrajectoryRequestHeader(nil), snapshot.TrajectoryRequestHeaders...)
	sort.SliceStable(headers, func(i, j int) bool { return headers[i].Seq < headers[j].Seq })
	index := -1
	for i, h := range headers {
		if h.Seq <= row.Source.Seq {
			index = i
		}
	}
	if index < 0 {
		return
	}
	raw, _ := json.MarshalIndent(headers[index].Header.Tools, "", "  ")
	after = string(raw)
	if index > 0 {
		raw, _ = json.MarshalIndent(headers[index-1].Header.Tools, "", "  ")
		before = string(raw)
	}
	return
}
func trajectorySimpleDiff(before, after string) string {
	if before == after {
		return ""
	}
	return diff.LineDiff(before, after)
}
func trajectoryUsage(snapshot session.ViewState, row trajRow) (session.TrajectoryRequest, map[string]float64, bool) {
	cumulative := map[string]float64{}
	var selected session.TrajectoryRequest
	found := false
	for _, request := range snapshot.TrajectoryRequests {
		if request.StartSeq > row.Source.Seq {
			continue
		}
		var usage map[string]float64
		if json.Unmarshal(request.Usage, &usage) == nil {
			for k, v := range usage {
				cumulative[k] += v
			}
		}
		if "request:"+request.ID == row.ID {
			selected = request
			found = true
		}
	}
	return selected, cumulative, found
}
func trajectoryProgramAvailable(source session.Message) bool {
	name := source.Body
	if source.Tool != nil {
		name = source.Tool.Name
	}
	var wire struct{ Data struct{ Name string } }
	_ = json.Unmarshal(source.Raw, &wire)
	if wire.Data.Name != "" {
		name = wire.Data.Name
	}
	return name == "run_code"
}
func trajectoryNativeDetail(c *ui.Context, m TrajectoryModel, row trajRow, all []trajRow, tab string, selected *string, fg, muted, field, accent ui.Color) (bool, string) {
	style := transcriptMarkdownStyle(fg, muted, field, accent, 13, darkSurface(muted))
	renderedImages := map[string]bool{}
	style.RenderImage = func(ctx *ui.Context, ref, alt string) bool {
		renderedImages[ref] = true
		ImageInlineView(ctx, ref, fg, muted, m.OnOpenImage)
		return true
	}
	source := row.Source
	switch tab {
	case "result":
		return trajectoryToolResultView(c, source, fg, muted, m.OnOpenImage)
	case "source":
		body := trajectorySource(row)
		trajectoryPayloadView(c, "source:"+row.ID, body, fg, muted)
		return true, body
	case "raw":
		body := strings.Join([]string{row.Full.Input, row.Full.Output, row.Full.Think, row.Full.Tools}, "\n")
		if source.Body != "" {
			body = source.Body
		}
		trajectoryPayloadView(c, "raw:"+row.ID, body, fg, muted)
		return true, body
	case "program":
		args := source.Args
		if source.Tool != nil {
			args = source.Tool.Arguments
		}
		program, present := toolCodeProgram(args)
		if !present {
			ui.Text(c, trajectoryText("没有已记录程序", "No recorded program")).TextColor(muted)
			return true, ""
		}
		root := ui.Column(c.Key("trajectory-program:" + row.ID)).FillWidth().Gap(6)
		jsonMode := ui.Local(root, "json", func() bool { return false })
		root.Children(func() {
			ui.Row(c).Gap(6).Children(func() {
				label := trajectoryText("JSON 参数", "JSON arguments")
				if *jsonMode {
					label = trajectoryText("程序源码", "Program source")
				}
				if ui.Button(c, label).Clicked() {
					*jsonMode = !*jsonMode
				}
				if ui.Button(c, trajectoryText("复制程序", "Copy program")).Clicked() {
					c.Services().WriteClipboard(program)
				}
			})
			if *jsonMode {
				trajectoryPayloadView(c, "program-json:"+row.ID, args, fg, muted)
			} else {
				markdown.ToolCodeBlock(c, "typescript", program, style, 0)
			}
		})
		return true, program
	case "history-diff":
		before, after := trajectoryHistoryPair(m.Snapshot, row)
		patch := trajectorySimpleDiff(before, after)
		if patch == "" {
			ui.Text(c, trajectoryText("已加载历史中没有变化", "No change in loaded history")).TextColor(muted)
		} else {
			diff.View(c, patch, diff.Style{FontSize: 12, FG: fg, Muted: muted, Add: ui.RGB(40, 160, 90), Del: ui.RGB(210, 70, 70), Hunk: accent})
		}
		return true, patch
	case "usage":
		request, cumulative, found := trajectoryUsage(m.Snapshot, row)
		if !found {
			ui.Text(c, trajectoryText("选择请求以查看用量", "Select a request to inspect its accounting")).TextColor(muted)
			return true, ""
		}
		body, _ := json.MarshalIndent(map[string]any{"request": json.RawMessage(request.Usage), "cumulativeThroughRequest": cumulative}, "", "  ")
		trajectoryPayloadView(c, "usage:"+row.ID, string(body), fg, muted)
		return true, string(body)
	case "overview", "preview":
		trajectoryLinks(c, m, row, all, selected)
		root := ui.Column(c.Key("trajectory-preview:" + row.ID)).FillWidth().Gap(6)
		thinking := ui.Local(root, "thinking", func() bool { return true })
		body := source.Body
		if body == "" {
			body = strings.Join([]string{row.Full.Input, row.Full.Output, row.Full.Tools}, "\n")
		}
		root.Children(func() {
			if source.Role == "user" {
				content := session.UserMessageContent(source)
				if content.Structured {
					userContentAttachments(c, content.Attachments, fg, muted, field, field, m.OnOpenImage)
					userReferenceText(c, content.Text, 13, fg, muted, nil, userReferenceFacts{Interactive: false, Occurrence: m.Snapshot.SelectedID + ":" + row.ID})
					for index, block := range content.ExtraBlocks {
						userExtraBlockView(c.Key(index), block, fg, muted)
					}
					return
				}
			}
			if painted, text := trajectoryToolResultView(c, source, fg, muted, m.OnOpenImage); painted {
				body = text
				return
			}
			if row.Full.Think != "" {
				if ui.Button(c, trajectoryText("思考", "Thinking")).Clicked() {
					*thinking = !*thinking
				}
				if *thinking {
					markdown.View(c, row.Full.Think, style)
				}
			}
			markdown.View(c, body, style)
			refs := session.ImageRefs(body)
			if source.ImageRef != "" {
				refs = append(refs, source.ImageRef)
			}
			refs = append(refs, trajectoryStructuredImageRefs(source)...)
			seen := map[string]bool{}
			for _, ref := range refs {
				if ref != "" && !seen[ref] && !renderedImages[ref] {
					seen[ref] = true
					ImageInlineView(c, ref, fg, muted, m.OnOpenImage)
				}
			}
		})
		return true, body
	case "timing":
		unix := ui.Local(c.Root(), "trajectory-unix-time", func() bool { return false })
		label := trajectoryText("Unix 秒", "Unix seconds")
		if *unix {
			label = trajectoryText("本地日期时间", "Local date/time")
		}
		if ui.Button(c, label).Clicked() {
			*unix = !*unix
		}
		body := trajectoryTimingDetail(row)
		if *unix {
			replace := func(ms float64) { body = strings.ReplaceAll(body, formatEventTime(ms), fmt.Sprintf("%.3f", ms/1000)) }
			if row.EventTime > 0 {
				replace(row.EventTime)
			}
			if row.Timing != nil {
				if row.Timing.StartedAt != nil {
					replace(*row.Timing.StartedAt)
				}
				if row.Timing.CompletedAt != nil {
					replace(*row.Timing.CompletedAt)
				}
			}
		}
		ui.Text(c, body).FontSize(12).TextColor(fg).Selectable()
		return true, body
	}
	return false, ""
}

// Canonical structured image blocks are projected only through content seats,
// never by interpreting arbitrary tool arguments as filesystem paths.
func trajectoryStructuredImageRefs(msg session.Message) []string {
	var root any
	if json.Unmarshal(msg.Raw, &root) != nil {
		return nil
	}
	var refs []string
	var visit func(any)
	visit = func(value any) {
		switch x := value.(type) {
		case []any:
			for _, v := range x {
				visit(v)
			}
		case map[string]any:
			if x["type"] == "image" {
				if a, ok := x["attachment"].(map[string]any); ok {
					if id, ok := a["attachmentId"].(string); ok && id != "" {
						refs = append(refs, "attachment:"+id)
					}
				}
			}
			for _, key := range []string{"data", "message", "result", "content"} {
				if v, ok := x[key]; ok {
					visit(v)
				}
			}
		}
	}
	visit(root)
	return refs
}

// Tool-result image blocks belong to the rendered result as well as its
// overview. Keep their canonical references live even before any Chat read,
// and leave text/foreign blocks in their original order beside those images.
func trajectoryToolResultView(c *ui.Context, source session.Message, fg, muted ui.Color, onOpen func(string)) (bool, string) {
	if source.Tool != nil && source.Tool.Result != nil {
		source = *source.Tool.Result
	}
	result, ok := terminalModelReadResult(&source)
	if !ok {
		return false, ""
	}
	content, ok := contextRawArray(result.content)
	if !ok {
		return false, ""
	}
	refs := make(map[int]string)
	for index, raw := range content {
		block, valid := terminalModelObject(raw)
		if !valid {
			continue
		}
		kind, _ := terminalModelString(block["type"])
		if kind != "image" {
			continue
		}
		attachment, valid := terminalModelObject(block["attachment"])
		if !valid {
			continue
		}
		if id, valid := terminalModelString(attachment["attachmentId"]); valid && id != "" {
			refs[index] = "attachment:" + id
		}
	}
	if len(refs) == 0 {
		return false, ""
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
		for index, raw := range content {
			if ref := refs[index]; ref != "" {
				ImageInlineView(c.Key(index), ref, fg, muted, onOpen)
				continue
			}
			block, _ := terminalModelObject(raw)
			kind, _ := terminalModelString(block["type"])
			text, valid := terminalModelString(block["text"])
			if kind != "text" || !valid {
				text = contextFormattedJSON(raw)
			}
			if text != "" {
				ui.Text(c.Key(index), text).FillWidth().MinWidth(0).Font(theme.RenderPreferences().Code.Family).FontSize(13).FixedLineHeight(20).TextColor(fg).Selectable()
			}
		}
	})
	return true, toolResultBlocksText(content, false)
}
