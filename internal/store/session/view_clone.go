package session

import (
	"encoding/json"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/dsh/events"
	"whaleisle/internal/store/sessioncost"
	"whaleisle/internal/workflowrun"
)

// cloneViewState returns a fully independent copy of in. Snapshot stores one
// clone and hands out another, so a caller that mutates any returned field
// (including nested rows, slices and pointers) can never reach the cache.
//
// This is deliberately an explicit per-field copy: a shallow value return
// aliased membership rows and a reflection-based deep copy aliased map state,
// both of which the package's contract tests reject.
func cloneViewState(in ViewState) ViewState { return cloneViewStateSchemas(in, true) }

// cloneFrameViewState is cloneViewState for a FrameSnapshot pass, whose view is
// read-only by contract. Request-header tool schemas are decoded once from the
// journal and never written by the store or the UI, yet every LLM request
// carries every tool's schema, so on a long transcript copying them dominated
// each frame (tmp/perf-20261029). The frame view shares them with the
// immutable cache; every other field is still copied.
func cloneFrameViewState(in ViewState) ViewState { return cloneViewStateSchemas(in, false) }

// cloneStoreViewCache clones ViewState for s.viewCache, preserving the heavy
// history/message slices that ensureViewCacheLocked has already cloned when
// their revisions changed.
func cloneStoreViewCache(in ViewState) ViewState {
	out := in
	out.SubagentCatalog = cloneSlice(in.SubagentCatalog)
	out.SidebarSubagentActivity = cloneSidebarSubagentActivity(in.SidebarSubagentActivity)
	out.ContextOccupancy = cloneContextOccupancy(in.ContextOccupancy)
	out.TurnFiles = cloneTurnFilesSnapshot(in.TurnFiles)
	out.WorkspaceLayout = cloneWorkspaceLayoutState(in.WorkspaceLayout)
	out.Workspaces = cloneWorkspaceRows(in.Workspaces)
	out.WorkspaceNavigation = cloneWorkspaceNavigationState(in.WorkspaceNavigation)
	out.QuestionProjection = cloneQuestionProjectionState(in.QuestionProjection)
	out.ArchiveConfirmation = cloneArchiveConfirmation(in.ArchiveConfirmation)
	out.SlashCommands = cloneCommandDescriptors(in.SlashCommands)
	out.QueueEditor = in.QueueEditor
	out.CostSources = cloneCostSources(in.CostSources)
	out.WorkspaceMembers = cloneStringBoolMap(in.WorkspaceMembers)
	out.Items = cloneItems(in.Items)
	out.MessageRatings = cloneStringStringMap(in.MessageRatings)
	out.ModelChoices = cloneModelChoiceMap(in.ModelChoices)
	out.ModelCatalogChoices = cloneSlice(in.ModelCatalogChoices)
	out.ModelEffortChoices = cloneModelChoiceMap(in.ModelEffortChoices)
	out.ModelSelection = cloneModelSelectionPtr(in.ModelSelection)
	out.ModelEffortLabels = cloneSlice(in.ModelEffortLabels)
	out.ModelLabels = cloneSlice(in.ModelLabels)
	out.PermissionLabels = cloneSlice(in.PermissionLabels)
	out.PlanModeControl = in.PlanModeControl
	out.AgentPresetLabels = cloneSlice(in.AgentPresetLabels)
	out.SkillCatalog = cloneSkillCatalog(in.SkillCatalog)
	out.SkillOptions = cloneSlice(in.SkillOptions)
	out.SkillHubHits = cloneSlice(in.SkillHubHits)
	out.SearchHitIDs = cloneStringBoolMap(in.SearchHitIDs)
	out.SearchResults = cloneSlice(in.SearchResults)
	out.PlanBoard = clonePlanBoardState(in.PlanBoard)
	out.HistoryNavigation = cloneHistoryNavigationView(in.HistoryNavigation)
	out.PinnedSessionIDs = cloneSlice(in.PinnedSessionIDs)
	out.ArchivedSessionIDs = cloneSlice(in.ArchivedSessionIDs)
	out.BandActions = cloneSlice(in.BandActions)
	return out
}

// cloneFramePassView creates a FrameSnapshot's read-only view from s.viewCache
// with a distinct top-level Messages slice header so overlapping frames never
// share slice backing arrays while avoiding per-frame deep copies of Raw bytes.
func cloneFramePassView(in ViewState) ViewState {
	out := in
	out.Messages = cloneSlice(in.Messages)
	return out
}

func cloneViewStateSchemas(in ViewState, schemas bool) ViewState {
	// Copy every value field, then replace each reference field below.
	out := in
	out.SubagentCatalog = cloneSlice(in.SubagentCatalog)
	out.SidebarSubagentActivity = cloneSidebarSubagentActivity(in.SidebarSubagentActivity)

	out.TrajectoryRequests = cloneTrajectoryRequests(in.TrajectoryRequests, schemas)
	out.TrajectoryRequestHeaders = cloneTrajectoryRequestHeaders(in.TrajectoryRequestHeaders, schemas)
	out.ContextOccupancy = cloneContextOccupancy(in.ContextOccupancy)
	out.TurnUsage = cloneTurnUsageMap(in.TurnUsage)
	out.TurnFiles = cloneTurnFilesSnapshot(in.TurnFiles)
	out.WorkflowRuns = cloneWorkflowRuns(in.WorkflowRuns)
	out.WorkspaceLayout = cloneWorkspaceLayoutState(in.WorkspaceLayout)
	out.Workspaces = cloneWorkspaceRows(in.Workspaces)
	out.WorkspaceNavigation = cloneWorkspaceNavigationState(in.WorkspaceNavigation)
	out.QuestionProjection = cloneQuestionProjectionState(in.QuestionProjection)
	out.ArchiveConfirmation = cloneArchiveConfirmation(in.ArchiveConfirmation)
	out.SlashCommands = cloneCommandDescriptors(in.SlashCommands)
	out.QueueEditor = in.QueueEditor // value-only fields
	out.CostSources = cloneCostSources(in.CostSources)
	out.WorkspaceMembers = cloneStringBoolMap(in.WorkspaceMembers)
	out.Items = cloneItems(in.Items)
	out.Messages = cloneMessages(in.Messages)
	out.MessageRatings = cloneStringStringMap(in.MessageRatings)
	out.ModelChoices = cloneModelChoiceMap(in.ModelChoices)
	out.ModelCatalogChoices = cloneSlice(in.ModelCatalogChoices)
	out.ModelEffortChoices = cloneModelChoiceMap(in.ModelEffortChoices)
	out.ModelSelection = cloneModelSelectionPtr(in.ModelSelection)
	out.ModelEffortLabels = cloneSlice(in.ModelEffortLabels)
	out.ModelLabels = cloneSlice(in.ModelLabels)
	out.PermissionLabels = cloneSlice(in.PermissionLabels)
	out.PlanModeControl = in.PlanModeControl // value-only fields
	out.AgentPresetLabels = cloneSlice(in.AgentPresetLabels)
	out.SkillCatalog = cloneSkillCatalog(in.SkillCatalog)
	out.SkillOptions = cloneSlice(in.SkillOptions)
	out.SkillHubHits = cloneSlice(in.SkillHubHits)
	out.SearchHitIDs = cloneStringBoolMap(in.SearchHitIDs)
	out.SearchResults = cloneSlice(in.SearchResults)
	out.PlanBoard = clonePlanBoardState(in.PlanBoard)
	out.HistoryNavigation = cloneHistoryNavigationView(in.HistoryNavigation)
	out.PinnedSessionIDs = cloneSlice(in.PinnedSessionIDs)
	out.ArchivedSessionIDs = cloneSlice(in.ArchivedSessionIDs)
	out.BandActions = cloneSlice(in.BandActions)
	return out
}

func cloneSlice[T any](in []T) []T {
	if in == nil {
		return nil
	}
	out := make([]T, len(in))
	copy(out, in)
	return out
}

func cloneRawMessage(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}
	return cloneSlice(in)
}

func cloneRawMessages(in []json.RawMessage) []json.RawMessage {
	if in == nil {
		return nil
	}
	out := make([]json.RawMessage, len(in))
	for i := range in {
		out[i] = cloneRawMessage(in[i])
	}
	return out
}

func cloneFloat64Ptr(in *float64) *float64 {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func cloneStringPtr(in *string) *string {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func cloneIntPtr(in *int) *int {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func cloneBoolPtr(in *bool) *bool {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func cloneStringBoolMap(in map[string]bool) map[string]bool {
	if in == nil {
		return nil
	}
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringUint64Map(in map[string]uint64) map[string]uint64 {
	if in == nil {
		return nil
	}
	out := make(map[string]uint64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStringFloat64Map(in map[string]float64) map[string]float64 {
	if in == nil {
		return nil
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// cloneJSONValue copies the container shapes produced by encoding/json, so a
// tool parameter tree never shares a map or slice with the cached snapshot.
func cloneJSONValue(in any) any {
	switch v := in.(type) {
	case nil:
		return nil
	case map[string]any:
		return cloneJSONObject(v)
	case map[string]string:
		return cloneStringStringMap(v)
	case map[string]float64:
		return cloneStringFloat64Map(v)
	case map[string]bool:
		return cloneStringBoolMap(v)
	case []any:
		if v == nil {
			return []any(nil)
		}
		out := make([]any, len(v))
		for i := range v {
			out[i] = cloneJSONValue(v[i])
		}
		return out
	case []string:
		return cloneSlice(v)
	case []float64:
		return cloneSlice(v)
	case []map[string]any:
		if v == nil {
			return []map[string]any(nil)
		}
		out := make([]map[string]any, len(v))
		for i := range v {
			out[i] = cloneJSONObject(v[i])
		}
		return out
	case json.RawMessage:
		return cloneRawMessage(v)
	case []byte:
		return cloneSlice(v)
	default:
		// Scalars and json.Number are immutable values.
		return in
	}
}

func cloneJSONObject(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneJSONValue(v)
	}
	return out
}

func cloneContextOccupancy(in *ContextOccupancy) *ContextOccupancy {
	if in == nil {
		return nil
	}
	out := *in
	if in.Breakdown != nil {
		breakdown := *in.Breakdown
		out.Breakdown = &breakdown
	}
	return &out
}

func cloneTurnUsageMap(in map[float64]TurnUsage) map[float64]TurnUsage {
	if in == nil {
		return nil
	}
	out := make(map[float64]TurnUsage, len(in))
	for seq, usage := range in {
		usage.CacheRead = cloneFloat64Ptr(usage.CacheRead)
		usage.CacheWrite = cloneFloat64Ptr(usage.CacheWrite)
		usage.Reasoning = cloneFloat64Ptr(usage.Reasoning)
		usage.Routes = cloneSlice(usage.Routes)
		out[seq] = usage
	}
	return out
}

func cloneTrajectoryRequests(in []TrajectoryRequest, schemas bool) []TrajectoryRequest {
	if in == nil {
		return nil
	}
	out := make([]TrajectoryRequest, len(in))
	for i := range in {
		out[i] = cloneTrajectoryRequest(in[i], schemas)
	}
	return out
}

func cloneTrajectoryRequest(in TrajectoryRequest, schemas bool) TrajectoryRequest {
	out := in
	out.Turn = cloneFloat64Ptr(in.Turn)
	out.Step = cloneFloat64Ptr(in.Step)
	out.StartedAt = cloneFloat64Ptr(in.StartedAt)
	out.CompletedAt = cloneFloat64Ptr(in.CompletedAt)
	out.ResultSeq = cloneFloat64Ptr(in.ResultSeq)
	out.ReplacementSeq = cloneFloat64Ptr(in.ReplacementSeq)
	if in.ProviderMetadata != nil {
		metadata := *in.ProviderMetadata
		out.ProviderMetadata = &metadata
	}
	out.Usage = cloneRawMessage(in.Usage)
	out.RequestConfig = cloneRawMessage(in.RequestConfig)
	out.Header = cloneTrajectoryRequestHeaderPtr(in.Header, schemas)
	out.RawOutput = cloneStringPtr(in.RawOutput)
	out.SummaryBlocks = cloneRawMessage(in.SummaryBlocks)
	out.RawOutputBlocks = cloneRawMessage(in.RawOutputBlocks)
	out.Error = cloneStringPtr(in.Error)
	out.Retries = cloneRetryAttempts(in.Retries)
	out.Attempts = cloneTrajectoryAttempts(in.Attempts)
	out.LiveAttempt = cloneTrajectoryLiveAttempt(in.LiveAttempt)
	out.Events = cloneRawMessages(in.Events)
	return out
}

func cloneRetryAttempts(in []RetryAttempt) []RetryAttempt {
	if in == nil {
		return nil
	}
	out := make([]RetryAttempt, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Failure = cloneRawMessage(in[i].Failure)
	}
	return out
}

func cloneTrajectoryAttempts(in []TrajectoryAttemptEvidence) []TrajectoryAttemptEvidence {
	if in == nil {
		return nil
	}
	out := make([]TrajectoryAttemptEvidence, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Raw = cloneRawMessage(in[i].Raw)
	}
	return out
}

func cloneTrajectoryLiveAttempt(in *TrajectoryLiveAttempt) *TrajectoryLiveAttempt {
	if in == nil {
		return nil
	}
	out := *in
	out.Usage = cloneStringFloat64Map(in.Usage)
	out.Stream = cloneRawMessages(in.Stream)
	return &out
}

func cloneTrajectoryRequestHeaders(in []TrajectoryRequestHeader, schemas bool) []TrajectoryRequestHeader {
	if in == nil {
		return nil
	}
	out := make([]TrajectoryRequestHeader, len(in))
	for i := range in {
		out[i] = cloneTrajectoryRequestHeader(in[i], schemas)
	}
	return out
}

func cloneTrajectoryRequestHeaderPtr(in *TrajectoryRequestHeader, schemas bool) *TrajectoryRequestHeader {
	if in == nil {
		return nil
	}
	out := cloneTrajectoryRequestHeader(*in, schemas)
	return &out
}

func cloneTrajectoryRequestHeader(in TrajectoryRequestHeader, schemas bool) TrajectoryRequestHeader {
	out := in
	out.Header = cloneRequestInspectionHeader(in.Header, schemas)
	return out
}

// schemas false shares each tool's Parameters map; see cloneFrameViewState.
func cloneRequestInspectionHeader(in RequestInspectionHeader, schemas bool) RequestInspectionHeader {
	out := in
	if in.Config != nil {
		config := *in.Config
		config.MaxTokens = cloneFloat64Ptr(in.Config.MaxTokens)
		config.ReasoningEffort = cloneStringPtr(in.Config.ReasoningEffort)
		config.Stop = cloneSlice(in.Config.Stop)
		out.Config = &config
	}
	if in.Tools != nil {
		out.Tools = make([]events.RequestHeaderDataHeaderToolsItem, len(in.Tools))
		for i := range in.Tools {
			out.Tools[i] = in.Tools[i]
			out.Tools[i].DeferLoading = cloneBoolPtr(in.Tools[i].DeferLoading)
			if schemas {
				out.Tools[i].Parameters = cloneJSONObject(in.Tools[i].Parameters)
			}
		}
	}
	return out
}

func cloneTurnFilesSnapshot(in TurnFilesSnapshot) TurnFilesSnapshot {
	out := in
	if in.Results == nil {
		return out
	}
	out.Results = make(map[TurnFileRequest]TurnFileResult, len(in.Results))
	for key, result := range in.Results {
		out.Results[key] = cloneTurnFileResult(result)
	}
	return out
}

func cloneTurnFileResult(in TurnFileResult) TurnFileResult {
	out := in
	if in.Summary != nil {
		summary := *in.Summary
		summary.Files = cloneSlice(in.Summary.Files)
		out.Summary = &summary
	}
	if in.Diff != nil {
		diff := *in.Diff
		diff.Before = cloneBoolPtr(in.Diff.Before)
		diff.After = cloneBoolPtr(in.Diff.After)
		diff.Coarse = cloneBoolPtr(in.Diff.Coarse)
		if in.Diff.Hunks != nil {
			diff.Hunks = make([]TurnDiffHunk, len(in.Diff.Hunks))
			for i := range in.Diff.Hunks {
				diff.Hunks[i] = in.Diff.Hunks[i]
				diff.Hunks[i].Lines = cloneSlice(in.Diff.Hunks[i].Lines)
			}
		}
		out.Diff = &diff
	}
	if in.Host != nil {
		host := *in.Host
		host.FileManager = cloneStringPtr(in.Host.FileManager)
		out.Host = &host
	}
	if in.Applications != nil {
		out.Applications = make([]TurnFileApplication, len(in.Applications))
		for i := range in.Applications {
			out.Applications[i] = in.Applications[i]
			out.Applications[i].Icon = cloneStringPtr(in.Applications[i].Icon)
		}
	}
	return out
}

func cloneWorkflowRuns(in []workflowrun.Run) []workflowrun.Run {
	if in == nil {
		return nil
	}
	out := make([]workflowrun.Run, len(in))
	for i := range in {
		out[i] = cloneWorkflowRun(in[i])
	}
	return out
}

func cloneWorkflowRun(in workflowrun.Run) workflowrun.Run {
	out := in
	if in.Phases != nil {
		out.Phases = make([]workflowrun.Phase, len(in.Phases))
		for i := range in.Phases {
			out.Phases[i] = in.Phases[i]
			out.Phases[i].Name = cloneStringPtr(in.Phases[i].Name)
			out.Phases[i].Members = cloneSlice(in.Phases[i].Members)
		}
	}
	out.Events = cloneRawMessages(in.Events)
	out.EventSeqs = cloneSlice(in.EventSeqs)
	return out
}

func cloneWorkspaceLayoutState(in WorkspaceLayoutState) WorkspaceLayoutState {
	out := in
	if in.SessionOrders != nil {
		out.SessionOrders = make(map[string][]string, len(in.SessionOrders))
		for key, order := range in.SessionOrders {
			out.SessionOrders[key] = cloneSlice(order)
		}
	}
	out.Collapsed = cloneStringBoolMap(in.Collapsed)
	return out
}

func cloneWorkspaceRows(in []WorkspaceRow) []WorkspaceRow {
	if in == nil {
		return nil
	}
	out := make([]WorkspaceRow, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].SessionIDs = cloneSlice(in[i].SessionIDs)
	}
	return out
}

func cloneWorkspaceNavigationState(in *WorkspaceNavigationState) *WorkspaceNavigationState {
	if in == nil {
		return nil
	}
	out := *in
	if in.Listing != nil {
		listing := *in.Listing
		listing.Crumbs = cloneSlice(in.Listing.Crumbs)
		listing.Entries = cloneSlice(in.Listing.Entries)
		out.Listing = &listing
	}
	return &out
}

func cloneQuestionProjectionState(in QuestionProjectionState) QuestionProjectionState {
	out := in
	out.Questions = cloneRawMessage(in.Questions)
	out.Inbox = cloneRawMessage(in.Inbox)
	out.QueueSeen = cloneStringUint64Map(in.QueueSeen)
	return out
}

func cloneArchiveConfirmation(in *ArchiveConfirmation) *ArchiveConfirmation {
	if in == nil {
		return nil
	}
	out := *in
	if in.Activity != nil {
		out.Activity = make([]ArchiveActivity, len(in.Activity))
		for i := range in.Activity {
			out.Activity[i] = in.Activity[i]
			out.Activity[i].Items = cloneSlice(in.Activity[i].Items)
		}
	}
	return &out
}

func cloneCostSources(in []sessioncost.Source) []sessioncost.Source {
	if in == nil {
		return nil
	}
	out := make([]sessioncost.Source, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Usage = cloneSessionCostProjection(in[i].Usage)
	}
	return out
}

func cloneSessionCostProjection(in *sessioncost.Projection) *sessioncost.Projection {
	if in == nil {
		return nil
	}
	out := *in
	if in.Models != nil {
		out.Models = make([]sessioncost.Route, len(in.Models))
		for i := range in.Models {
			out.Models[i] = in.Models[i]
			out.Models[i].Peak = cloneSessionCostBucket(in.Models[i].Peak)
			out.Models[i].OffPeak = cloneSessionCostBucket(in.Models[i].OffPeak)
		}
	}
	return &out
}

func cloneSessionCostBucket(in *sessioncost.Bucket) *sessioncost.Bucket {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneItems(in []Item) []Item {
	if in == nil {
		return nil
	}
	out := make([]Item, len(in))
	for i := range in {
		out[i] = in[i]
		if in[i].Presentation != nil {
			presentation := *in[i].Presentation
			out[i].Presentation = &presentation
		}
		if in[i].SubagentIdentity != nil {
			identity := *in[i].SubagentIdentity
			out[i].SubagentIdentity = &identity
		}
		out[i].AgentMetrics = cloneAgentMetrics(in[i].AgentMetrics)
		out[i].BilledUsage = cloneSessionCostProjection(in[i].BilledUsage)
	}
	return out
}

func cloneAgentMetrics(in AgentMetrics) AgentMetrics {
	out := in
	out.Tokens = cloneFloat64Ptr(in.Tokens)
	if in.Timing != nil {
		timing := *in.Timing
		timing.Since = cloneFloat64Ptr(in.Timing.Since)
		timing.Through = cloneFloat64Ptr(in.Timing.Through)
		out.Timing = &timing
	}
	return out
}

func cloneMessages(in []Message) []Message {
	if in == nil {
		return nil
	}
	out := make([]Message, len(in))
	for i := range in {
		out[i] = cloneMessage(in[i])
	}
	return out
}

func cloneMessage(in Message) Message {
	out := in
	out.AutomaticCompaction = cloneManualCompactionView(in.AutomaticCompaction)
	if in.Retry != nil {
		retry := RetryStatus{Attempts: cloneRetryAttempts(in.Retry.Attempts)}
		out.Retry = &retry
	}
	out.TurnDelivery = cloneTurnDelivery(in.TurnDelivery)
	if in.TrajectoryTime != nil {
		timing := *in.TrajectoryTime
		timing.StartedAt = cloneFloat64Ptr(in.TrajectoryTime.StartedAt)
		timing.CompletedAt = cloneFloat64Ptr(in.TrajectoryTime.CompletedAt)
		timing.FirstTokenAt = cloneFloat64Ptr(in.TrajectoryTime.FirstTokenAt)
		out.TrajectoryTime = &timing
	}
	if in.Workflow != nil {
		workflow := cloneWorkflowRun(*in.Workflow)
		out.Workflow = &workflow
	}
	out.Submission = clonePendingSubmission(in.Submission)
	out.Tool = cloneToolCallView(in.Tool)
	out.Command = cloneCommandView(in.Command)
	out.ConversationTurn = cloneFloat64Ptr(in.ConversationTurn)
	out.ConversationStep = cloneFloat64Ptr(in.ConversationStep)
	out.ContentBlock = cloneRawMessage(in.ContentBlock)
	out.Raw = cloneRawMessage(in.Raw)
	out.Files = cloneSlice(in.Files)
	if in.Web != nil {
		web := *in.Web
		web.Sources = cloneSlice(in.Web.Sources)
		out.Web = &web
	}
	return out
}

func cloneMessagePtr(in *Message) *Message {
	if in == nil {
		return nil
	}
	out := cloneMessage(*in)
	return &out
}

func cloneManualCompactionView(in *ManualCompactionView) *ManualCompactionView {
	if in == nil {
		return nil
	}
	out := *in
	out.SummaryEventSeq = cloneFloat64Ptr(in.SummaryEventSeq)
	out.ShadowedItemCount = cloneIntPtr(in.ShadowedItemCount)
	out.ShadowedTokenCount = cloneIntPtr(in.ShadowedTokenCount)
	return &out
}

func cloneTurnDelivery(in *TurnDelivery) *TurnDelivery {
	if in == nil {
		return nil
	}
	out := *in
	if in.Changes != nil {
		changes := *in.Changes
		out.Changes = &changes
	}
	out.Files = cloneSlice(in.Files)
	return &out
}

func clonePendingSubmission(in *PendingSubmission) *PendingSubmission {
	if in == nil {
		return nil
	}
	out := *in
	out.Attachments = cloneSlice(in.Attachments)
	return &out
}

func cloneToolCallView(in *ToolCallView) *ToolCallView {
	if in == nil {
		return nil
	}
	out := *in
	out.Call = cloneMessagePtr(in.Call)
	out.Result = cloneMessagePtr(in.Result)
	out.Events = cloneMessages(in.Events)
	if in.Children != nil {
		out.Children = make([]*ToolCallView, len(in.Children))
		for i := range in.Children {
			out.Children[i] = cloneToolCallView(in.Children[i])
		}
	}
	return &out
}

func cloneCommandView(in *CommandView) *CommandView {
	if in == nil {
		return nil
	}
	out := *in
	out.Compaction = cloneManualCompactionView(in.Compaction)
	out.SourceEventSeq = cloneFloat64Ptr(in.SourceEventSeq)
	return &out
}

func cloneModelChoiceMap(in map[string]api.ModelCatalogDefault) map[string]api.ModelCatalogDefault {
	if in == nil {
		return nil
	}
	out := make(map[string]api.ModelCatalogDefault, len(in))
	for key, choice := range in {
		out[key] = cloneModelSelection(choice)
	}
	return out
}

func cloneModelSelectionPtr(in *api.ModelCatalogDefault) *api.ModelCatalogDefault {
	if in == nil {
		return nil
	}
	out := cloneModelSelection(*in)
	return &out
}

func cloneSkillCatalog(in SkillCatalog) SkillCatalog {
	out := in
	if in.Skills != nil {
		out.Skills = make([]api.SkillInventorySnapshotSkillsItem, len(in.Skills))
		for i := range in.Skills {
			out.Skills[i] = in.Skills[i]
			out.Skills[i].WhenToUse = cloneStringPtr(in.Skills[i].WhenToUse)
			out.Skills[i].Groups = cloneSlice(in.Skills[i].Groups)
			out.Skills[i].Path = cloneStringPtr(in.Skills[i].Path)
			out.Skills[i].Directory = cloneStringPtr(in.Skills[i].Directory)
		}
	}
	out.Errors = cloneStringStringMap(in.Errors)
	return out
}

func clonePlanBoardState(in PlanBoard) PlanBoard {
	out := PlanBoard{PlanMode: in.PlanMode, QueueActions: in.QueueActions}
	if in.goalReadToken != nil {
		token := *in.goalReadToken
		out.goalReadToken = &token
	}
	if in.Goal != nil {
		goal := *in.Goal
		out.Goal = &goal
	}
	out.Todos = cloneSlice(in.Todos)
	if in.Queue != nil {
		out.Queue = make([]PlanQueueItem, len(in.Queue))
		for i := range in.Queue {
			out.Queue[i] = in.Queue[i]
			out.Queue[i].Attachments = cloneSlice(in.Queue[i].Attachments)
		}
	}
	if in.Pending != nil {
		out.Pending = make([]PendingSubmission, len(in.Pending))
		for i := range in.Pending {
			out.Pending[i] = in.Pending[i]
			out.Pending[i].Attachments = cloneSlice(in.Pending[i].Attachments)
		}
	}
	return out
}

func cloneHistoryNavigationView(in HistoryNavigationView) HistoryNavigationView {
	out := in
	out.Turns = cloneSlice(in.Turns)
	return out
}
