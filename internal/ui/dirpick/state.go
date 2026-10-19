package dirpick

import (
	"strings"
	"whaleisle/internal/dsh/api"
)

// BrowserState mirrors DirectoryBrowser.tsx's resident Miller view and drafts.
// Listings are immutable Host replies; editable values are copied for painting.
type BrowserState struct {
	Parent, Child                        *api.DirectoryListing
	Selected                             *api.DirectoryListingCrumbsItem
	PathDraft, FolderDraft               *string
	ShowHidden, Creating, SlowScan       bool
	CreateError                          string
	ScannedDirectory, ScannedPath        string
	PreviewSuspended                     bool
	RetrySelection                       *api.DirectoryListingCrumbsItem
	FocusPath, FocusEdit, FocusSelection bool
	PathHadFocus                         bool
	ScrolledChild, ScrolledCrumb         string
	ScanWindow                           uint64
	FailureToast, CreateToast            string
	Revision                             uint64
}

func sameDraft(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

// MergeUI applies only changes made by this render's controls. Async Host
// replies own the panes, request state and errors and cannot be overwritten by
// a stale render clone. A submission that started meanwhile owns its draft.
func (s *BrowserState) MergeUI(before, edited BrowserState) {
	if before.ShowHidden != edited.ShowHidden {
		s.ShowHidden = edited.ShowHidden
	}
	if !sameDraft(before.FolderDraft, edited.FolderDraft) && !s.Creating && sameDraft(s.FolderDraft, before.FolderDraft) {
		s.FolderDraft = edited.Clone().FolderDraft
		if edited.CreateError == "" && s.CreateError == before.CreateError {
			s.CreateError = ""
		}
	}
	if s.Revision == before.Revision {
		if before.PathHadFocus != edited.PathHadFocus {
			s.PathHadFocus = edited.PathHadFocus
		}
		if before.FocusPath != edited.FocusPath {
			s.FocusPath = edited.FocusPath
		}
		if before.FocusEdit != edited.FocusEdit {
			s.FocusEdit = edited.FocusEdit
		}
		if before.FocusSelection != edited.FocusSelection {
			s.FocusSelection = edited.FocusSelection
		}
	}
	if before.ScrolledChild != edited.ScrolledChild && s.Child != nil && s.Child.Path == edited.ScrolledChild {
		s.ScrolledChild = edited.ScrolledChild
	}
	if before.ScrolledCrumb != edited.ScrolledCrumb && s.Current() != nil && s.Current().Path == edited.ScrolledCrumb {
		s.ScrolledCrumb = edited.ScrolledCrumb
	}
	if before.FailureToast != edited.FailureToast && s.FailureToast == before.FailureToast {
		s.FailureToast = edited.FailureToast
	}
	if before.CreateToast != edited.CreateToast && s.CreateToast == before.CreateToast {
		s.CreateToast = edited.CreateToast
	}
}

func (s BrowserState) Clone() BrowserState {
	if s.PathDraft != nil {
		value := *s.PathDraft
		s.PathDraft = &value
	}
	if s.FolderDraft != nil {
		value := *s.FolderDraft
		s.FolderDraft = &value
	}
	return s
}

func (s BrowserState) Current() *api.DirectoryListing {
	if s.Child != nil {
		return s.Child
	}
	return s.Parent
}

func (s BrowserState) TargetPath() string {
	if s.Selected != nil {
		return s.Selected.Path
	}
	if s.Parent != nil {
		return s.Parent.Path
	}
	return ""
}

func (s BrowserState) TargetName(home string) string {
	if s.Parent != nil && s.TargetPath() == s.Parent.Home {
		return home
	}
	if s.Selected != nil {
		return s.Selected.Name
	}
	if s.Parent == nil {
		return ""
	}
	if len(s.Parent.Crumbs) > 0 {
		return s.Parent.Crumbs[len(s.Parent.Crumbs)-1].Name
	}
	return s.Parent.Path
}

func Separator(listing *api.DirectoryListing) string {
	if listing != nil && strings.Contains(listing.Home, "\\") {
		return "\\"
	}
	return "/"
}

func DirectoryPart(listing *api.DirectoryListing, draft string) string {
	cut := strings.LastIndex(draft, "/")
	if Separator(listing) == "\\" {
		cut = max(cut, strings.LastIndex(draft, "\\"))
	}
	if cut < 0 {
		return ""
	}
	return draft[:cut+1]
}

// Prefix filters only the last pane, and only while it answers the draft.
func (s BrowserState) Prefix() (string, bool) {
	current := s.Current()
	if current == nil || s.PathDraft == nil {
		return "", false
	}
	directory := DirectoryPart(current, *s.PathDraft)
	if directory == "" {
		return "", false
	}
	level := current.Path
	if !strings.HasSuffix(level, Separator(current)) {
		level += Separator(current)
	}
	if directory != level && !(directory == s.ScannedDirectory && current.Path == s.ScannedPath) {
		return "", false
	}
	return (*s.PathDraft)[len(directory):], true
}

// The selected row survives filtering; a prefix miss restores the full level.
func VisibleEntries(entries []api.DirectoryListingCrumbsItem, selected string, showHidden bool, prefix string) []api.DirectoryListingCrumbsItem {
	needle := strings.ToLower(prefix)
	matches := func(entry api.DirectoryListingCrumbsItem) bool {
		return (showHidden || !entry.Hidden || strings.HasPrefix(needle, ".")) && strings.HasPrefix(strings.ToLower(entry.Name), needle)
	}
	narrow := false
	if needle != "" {
		for _, entry := range entries {
			if matches(entry) {
				narrow = true
				break
			}
		}
	}
	var visible []api.DirectoryListingCrumbsItem
	for _, entry := range entries {
		if entry.Path == selected || narrow && matches(entry) || !narrow && (showHidden || !entry.Hidden) {
			visible = append(visible, entry)
		}
	}
	return visible
}

func MatchEntry(parent api.DirectoryListing, path string) *api.DirectoryListingCrumbsItem {
	for _, entry := range parent.Entries {
		match := entry.Path == path
		if Separator(&parent) == "\\" {
			match = strings.EqualFold(entry.Path, path)
		}
		if match {
			copy := entry
			return &copy
		}
	}
	return nil
}
