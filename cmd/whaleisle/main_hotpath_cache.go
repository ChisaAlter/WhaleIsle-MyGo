package main

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"whaleisle/internal/errcopy"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/credentials"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/theme"
)

type mainViewWritebackSnapshot struct {
	layout               frame.LayoutSizes
	draft                string
	editMessageSeq       float64
	feedbackCommentMsgID string
	feedbackCommentDraft string
	modelPick            string
	permissionPick       string
	agentPresetPick      string
	thinkingExpanded     map[string]bool
	thinkingExpandedNil  bool
	renameSessionID      string
	renameDraft          string
	sessionSearch        string
	fileTabs             files.Tabs
	gitWorkbench         diffsurface.Workbench
	fileExpanded         map[string]bool
	fileExpandedNil      bool
	selectedFile         string
	fileFilter           string
	filePreview          string
	filePreviewErr       string
	filePreviewList      ui.ListState
	fileEditing          bool
	fileEditDraft        string
	fileEditStatus       string
	fileEditViewMode     files.EditViewMode
	fileEditSplit        float32
	rightPaneTab         string
	agentsPromptDraft    string
	goalDraft            string
	diffPath             string
	diffText             string
	diffErr              string
	planBoardOpen        bool
	todoPanelCollapsed   bool
	chatPaneTab          string
	trajSearch           string
	trajSelectedID       string
	trajActualDur        bool
	trajTurnsFold        bool
	trajCallsFold        bool
	trajDetailTab        string
	imageLightboxPath    string
	gitCommitOpen        bool
	gitCommitDraft       string
	gitPROpen            bool
}

type composerWritebackSnapshot struct {
	suggest      ui.ListState
	skillsList   ui.ListState
	skillsOpen   bool
	skillsFilter string
	skillCreate  string
	chipEdit     composer.ChipEdit
}

func cloneBoolMapForDiff(m map[string]bool) (map[string]bool, bool) {
	if m == nil {
		return nil, true
	}
	return maps.Clone(m), false
}

func boolMapChanged(cur, start map[string]bool, startNil bool) bool {
	if (cur == nil) != startNil {
		return true
	}
	return !maps.Equal(cur, start)
}

func captureComposerWriteback(suggest, skillsList ui.ListState, skillsOpen bool, skillsFilter, skillCreate string, chipEdit composer.ChipEdit) composerWritebackSnapshot {
	return composerWritebackSnapshot{
		suggest:      suggest,
		skillsList:   skillsList,
		skillsOpen:   skillsOpen,
		skillsFilter: skillsFilter,
		skillCreate:  skillCreate,
		chipEdit:     chipEdit,
	}
}

func (s *mainState) commitComposerWritebackLocked(start, cur composerWritebackSnapshot) {
	if !reflect.DeepEqual(cur.suggest, start.suggest) {
		s.composerSuggest = cur.suggest
	}
	if !reflect.DeepEqual(cur.skillsList, start.skillsList) {
		s.composerSkillsList = cur.skillsList
	}
	if cur.skillsOpen != start.skillsOpen {
		s.composerSkillsOpen = cur.skillsOpen
	}
	if cur.skillsFilter != start.skillsFilter {
		s.composerSkillsFilter = cur.skillsFilter
	}
	if cur.skillCreate != start.skillCreate {
		s.skillCreateName = cur.skillCreate
	}
	if !reflect.DeepEqual(cur.chipEdit, start.chipEdit) {
		s.composerChipEdit = cur.chipEdit
	}
}

func (s *mainState) commitMainViewWritebackLocked(start, cur mainViewWritebackSnapshot) string {
	bottom := s.layout.Bottom
	curLayoutCheck := cur.layout
	curLayoutCheck.Bottom = start.layout.Bottom
	if curLayoutCheck != start.layout {
		curLayoutWrite := cur.layout
		curLayoutWrite.Bottom = bottom
		s.layout = curLayoutWrite
	}

	if cur.draft != start.draft {
		s.draft = cur.draft
	}
	if cur.editMessageSeq != start.editMessageSeq {
		s.editMessageSeq = cur.editMessageSeq
	}
	if cur.feedbackCommentMsgID != start.feedbackCommentMsgID {
		s.feedbackCommentMsgID = cur.feedbackCommentMsgID
	}
	if cur.feedbackCommentDraft != start.feedbackCommentDraft {
		s.feedbackCommentDraft = cur.feedbackCommentDraft
	}
	if cur.modelPick != start.modelPick {
		s.modelPick = cur.modelPick
	}
	if cur.permissionPick != start.permissionPick {
		s.permissionPick = cur.permissionPick
	}
	if cur.agentPresetPick != start.agentPresetPick {
		s.agentPresetPick = cur.agentPresetPick
	}
	if boolMapChanged(cur.thinkingExpanded, start.thinkingExpanded, start.thinkingExpandedNil) {
		s.thinkingExpanded = cur.thinkingExpanded
	}
	if cur.renameSessionID != start.renameSessionID {
		s.renameSessionID = cur.renameSessionID
	}
	if cur.renameDraft != start.renameDraft {
		s.renameDraft = cur.renameDraft
	}
	if cur.sessionSearch != start.sessionSearch {
		s.sessionSearch = cur.sessionSearch
	}
	if !reflect.DeepEqual(cur.fileTabs, start.fileTabs) {
		s.fileTabs = cur.fileTabs
	}
	if !reflect.DeepEqual(cur.gitWorkbench, start.gitWorkbench) {
		s.gitWorkbench = cur.gitWorkbench
	}
	if boolMapChanged(cur.fileExpanded, start.fileExpanded, start.fileExpandedNil) {
		s.fileExpanded = cur.fileExpanded
	}
	if cur.selectedFile != start.selectedFile {
		s.selectedFile = cur.selectedFile
	}
	if cur.fileFilter != start.fileFilter {
		s.fileFilter = cur.fileFilter
	}
	if cur.filePreview != start.filePreview {
		s.filePreview = cur.filePreview
	}
	if cur.filePreviewErr != start.filePreviewErr {
		s.filePreviewErr = cur.filePreviewErr
	}
	if !reflect.DeepEqual(cur.filePreviewList, start.filePreviewList) {
		s.filePreviewList = cur.filePreviewList
	}
	if cur.fileEditing != start.fileEditing {
		s.fileEditing = cur.fileEditing
	}
	if cur.fileEditDraft != start.fileEditDraft {
		s.fileEditDraft = cur.fileEditDraft
	}
	if cur.fileEditStatus != start.fileEditStatus {
		s.fileEditStatus = cur.fileEditStatus
	}
	if cur.fileEditViewMode != start.fileEditViewMode {
		s.fileEditViewMode = cur.fileEditViewMode
	}
	if cur.fileEditSplit != start.fileEditSplit {
		s.fileEditSplit = cur.fileEditSplit
	}
	if cur.rightPaneTab != start.rightPaneTab {
		s.rightPaneTab = cur.rightPaneTab
	}
	if cur.agentsPromptDraft != start.agentsPromptDraft {
		s.agentsPromptDraft = cur.agentsPromptDraft
	}
	if cur.goalDraft != start.goalDraft {
		s.goalDraft = cur.goalDraft
	}
	if cur.diffPath != start.diffPath {
		s.diffPath = cur.diffPath
	}
	if cur.diffText != start.diffText {
		s.diffText = cur.diffText
	}
	if cur.diffErr != start.diffErr {
		s.diffErr = cur.diffErr
	}
	if cur.planBoardOpen != start.planBoardOpen {
		s.planBoardOpen = cur.planBoardOpen
	}
	if cur.todoPanelCollapsed != start.todoPanelCollapsed {
		s.todoPanelCollapsed = cur.todoPanelCollapsed
	}
	previousChatPaneTab := s.chatPaneTab
	if cur.chatPaneTab != start.chatPaneTab {
		s.chatPaneTab = cur.chatPaneTab
	}
	if cur.trajSearch != start.trajSearch {
		s.trajSearch = cur.trajSearch
	}
	if cur.trajSelectedID != start.trajSelectedID {
		s.trajSelectedID = cur.trajSelectedID
	}
	if cur.trajActualDur != start.trajActualDur {
		s.trajActualDur = cur.trajActualDur
	}
	if cur.trajTurnsFold != start.trajTurnsFold {
		s.trajTurnsFold = cur.trajTurnsFold
	}
	if cur.trajCallsFold != start.trajCallsFold {
		s.trajCallsFold = cur.trajCallsFold
	}
	if cur.trajDetailTab != start.trajDetailTab {
		s.trajDetailTab = cur.trajDetailTab
	}
	if cur.imageLightboxPath != start.imageLightboxPath {
		s.imageLightboxPath = cur.imageLightboxPath
	}
	if cur.gitCommitOpen != start.gitCommitOpen {
		s.gitCommitOpen = cur.gitCommitOpen
	}
	if cur.gitCommitDraft != start.gitCommitDraft {
		s.gitCommitDraft = cur.gitCommitDraft
	}
	if cur.gitPROpen != start.gitPROpen {
		s.gitPROpen = cur.gitPROpen
	}
	return previousChatPaneTab
}

func (s *mainState) commitGitWorkbenchIfChanged(start, cur diffsurface.Workbench) {
	if reflect.DeepEqual(cur, start) {
		return
	}
	s.mu.Lock()
	s.gitWorkbench = cur
	s.mu.Unlock()
}

// ensureCredentialsDraftSeededLocked triggers asynchronous credential loading
// outside s.mu and the UI frame so disk/keyring I/O never stalls painting.
// REQUIRES: s.mu held.
func (s *mainState) ensureCredentialsDraftSeededLocked() {
	if s.credsDraft.seeded || s.credsDraft.loadFailed || s.credsSeedLoading {
		return
	}
	s.credsSeedGen++
	gen := s.credsSeedGen
	s.credsSeedLoading = true
	ctx := s.lifecycleContextLocked()
	dir := paths.UserDataDir()
	go s.loadCredentialsDraftAsync(ctx, gen, dir)
}

func (s *mainState) ensureCredentialsDraftSeeded() {
	s.mu.Lock()
	s.ensureCredentialsDraftSeededLocked()
	s.mu.Unlock()
}

func (s *mainState) loadCredentialsDraftAsync(ctx context.Context, gen uint64, dir string) {
	if ctx.Err() != nil {
		s.mu.Lock()
		if s.credsSeedGen == gen {
			s.credsSeedLoading = false
		}
		s.mu.Unlock()
		return
	}
	editor, err := credentials.OpenSharedProfile(dir)
	s.mu.Lock()
	if s.credsSeedGen != gen {
		s.mu.Unlock()
		return
	}
	s.credsSeedLoading = false
	if ctx.Err() != nil || s.credsDraft.seeded {
		s.mu.Unlock()
		return
	}
	if err != nil {
		s.credsSaveErr = errcopy.Sentence(err)
		s.credsDraft.loadFailed = true
	} else {
		s.credsDraft = credentialDraftFromEditor(editor)
		s.credsSaveErr = s.credsDraft.writeAccessErr
	}
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}

func (s *mainState) cachedWallpaperChoices(active bool) []string {
	s.mu.Lock()
	if !active {
		s.wallpaperChoicesReady = false
		s.wallpaperChoicesLoading = false
		s.wallpaperChoicesGen++
		s.mu.Unlock()
		return nil
	}
	if !s.wallpaperChoicesReady && !s.wallpaperChoicesLoading {
		s.wallpaperChoicesGen++
		gen := s.wallpaperChoicesGen
		s.wallpaperChoicesLoading = true
		ctx := s.lifecycleContextLocked()
		dataDir := paths.UserDataDir()
		go s.loadWallpaperChoicesAsync(ctx, gen, dataDir)
	}
	choices := append([]string(nil), s.wallpaperChoicesCache...)
	s.mu.Unlock()
	return choices
}

func (s *mainState) loadWallpaperChoicesAsync(ctx context.Context, gen uint64, dataDir string) {
	if ctx.Err() != nil {
		s.mu.Lock()
		if s.wallpaperChoicesGen == gen {
			s.wallpaperChoicesLoading = false
		}
		s.mu.Unlock()
		return
	}
	exeDir, _ := os.Executable()
	choices := theme.ListWallpapers(theme.WallpaperGalleryDirs(dataDir, filepath.Dir(exeDir))...)
	s.mu.Lock()
	if s.wallpaperChoicesGen != gen {
		s.mu.Unlock()
		return
	}
	s.wallpaperChoicesLoading = false
	if ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	s.wallpaperChoicesCache = choices
	s.wallpaperChoicesReady = true
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}
