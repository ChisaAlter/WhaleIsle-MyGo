package main

import (
	"errors"
	"whaleisle/internal/errcopy"

	"whaleisle/internal/credentials"
	"whaleisle/internal/paths"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

type credentialsDraft struct {
	apiKey, baseURL, githubToken, remoteToken, remoteRelayToken string
	seeded, loadFailed, saving, conflicted                      bool
	editor                                                      *credentials.Store
	writeAccessErr                                              string
}

func (d credentialsDraft) fields() credentials.Fields {
	return credentials.Fields{APIKey: d.apiKey, BaseURL: d.baseURL, GithubToken: d.githubToken, RemoteToken: d.remoteToken, RemoteRelayToken: d.remoteRelayToken}
}

func credentialDraftFromEditor(editor *credentials.Store) credentialsDraft {
	f := editor.Fields()
	draft := credentialsDraft{apiKey: f.APIKey, baseURL: f.BaseURL, githubToken: f.GithubToken, remoteToken: f.RemoteToken, remoteRelayToken: f.RemoteRelayToken, seeded: true, editor: editor}
	if err := editor.WriteAccessError(); err != nil {
		draft.writeAccessErr = errcopy.Sentence(err)
	}
	return draft
}

func (s *mainState) credentialsCanSave() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.credsDraft.seeded && s.credsDraft.editor != nil && s.credsDraft.writeAccessErr == "" && !s.credsDraft.saving && !s.credsDraft.conflicted && !s.credsSeedLoading
}

func (s *mainState) credentialsLoaded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.credsDraft.seeded && s.credsDraft.editor != nil
}

// keepCredentialEdits applies freshly read values only to fields that have not
// changed from the supplied comparison point. It also preserves continued typing.
func keepCredentialEdits(current *credentialsDraft, comparison credentials.Fields, fresh credentialsDraft) {
	for _, field := range []struct {
		current     *string
		base, saved string
	}{
		{&current.apiKey, comparison.APIKey, fresh.apiKey},
		{&current.baseURL, comparison.BaseURL, fresh.baseURL},
		{&current.githubToken, comparison.GithubToken, fresh.githubToken},
		{&current.remoteToken, comparison.RemoteToken, fresh.remoteToken},
		{&current.remoteRelayToken, comparison.RemoteRelayToken, fresh.remoteRelayToken},
	} {
		if *field.current == field.base {
			*field.current = field.saved
		}
	}
}

// retryCredentialsLoad is an explicit file reload. Existing local edits are
// retained against a new baseline and require a separate Save gesture.
func (s *mainState) retryCredentialsLoad() {
	s.mu.Lock()
	if s.credsDraft.saving || s.credsSeedLoading {
		s.mu.Unlock()
		return
	}
	if s.credsDraft.editor == nil || !s.credsDraft.seeded {
		s.credsDraft.loadFailed = false
		s.ensureCredentialsDraftSeededLocked()
		s.mu.Unlock()
		return
	}
	previous := s.credsDraft.editor.Fields()
	s.credsSeedGen++
	gen, dir, ctx := s.credsSeedGen, paths.UserDataDir(), s.lifecycleContextLocked()
	s.credsSeedLoading = true
	s.mu.Unlock()
	go func() {
		editor, err := credentials.OpenSharedProfile(dir)
		s.mu.Lock()
		if gen != s.credsSeedGen {
			s.mu.Unlock()
			return
		}
		s.credsSeedLoading = false
		if ctx.Err() == nil {
			if err != nil {
				s.credsSaveErr = errcopy.Sentence(err)
			} else {
				fresh := credentialDraftFromEditor(editor)
				keepCredentialEdits(&s.credsDraft, previous, fresh)
				s.credsDraft.editor = editor
				s.credsDraft.writeAccessErr = fresh.writeAccessErr
				s.credsDraft.conflicted, s.credsDraft.loadFailed = false, false
				s.credsSaveErr = s.credsDraft.writeAccessErr
				s.credsLastAction = "已读取最新凭据并保留本页编辑。 / Latest credentials loaded; local edits retained."
			}
		}
		win := s.win
		s.mu.Unlock()
		if win != nil {
			win.Invalidate()
		}
	}()
}

// saveCredentials captures the edit on the UI thread; disk/keyring and Host
// work runs outside the UI callback and state lock. REQUIRES: s.mu held.
func (s *mainState) saveCredentials() {
	if !s.credsDraft.seeded || s.credsDraft.editor == nil || s.credsDraft.saving || s.credsDraft.conflicted || s.credsSeedLoading {
		return
	}
	if s.credsDraft.writeAccessErr != "" {
		s.credsSaveErr = s.credsDraft.writeAccessErr
		return
	}
	submitted, gen, ctx := s.credsDraft, s.credsSeedGen, s.lifecycleContextLocked()
	apiKeyChanged := submitted.apiKey != submitted.editor.Fields().APIKey
	s.credsDraft.saving = true
	s.credsSaveErr, s.credsLastAction = "", ""
	go func() {
		s.dshMu.Lock()
		client := s.dshClient
		s.dshMu.Unlock()
		committed, err := submitted.editor.CommitFields(submitted.fields())
		var syncErr error
		synced := false
		if err == nil && apiKeyChanged {
			s.dshMu.Lock()
			current := client != nil && s.dshClient == client
			s.dshMu.Unlock()
			if current {
				syncErr = settingsstore.SyncAPIKeyToHarness(ctx, client, committed.APIKey)
				synced = syncErr == nil
			}
		}
		s.mu.Lock()
		if err == nil {
			// Applying an accepted save does not depend on whether its editor
			// is still visible; only the draft feedback is generation-bound.
			s.applyRemoteCredentials(committed)
		}
		if gen != s.credsSeedGen || s.credsDraft.editor != submitted.editor || !s.credsDraft.seeded {
			s.mu.Unlock()
			return
		}
		s.credsDraft.saving = false
		if err != nil {
			if errors.Is(err, credentials.ErrConflict) {
				s.credsDraft.conflicted = true
				s.credsSaveErr = "凭据已在其他窗口更改，编辑内容已保留。请重新读取文件、核对后保存。 / Credentials changed in another window. Reload the file, review your retained edits, then save."
			} else {
				s.credsSaveErr = errcopy.Sentence(err)
			}
		} else {
			keepCredentialEdits(&s.credsDraft, submitted.fields(), credentialDraftFromEditor(submitted.editor))
			s.credsLastAction = i18n.T(i18n.KeySettingsCredSaved)
			if synced {
				s.credsLastAction += " · credentials/set"
			}
			if syncErr != nil {
				s.credsHarnessNote = errcopy.Sentence(syncErr)
			} else {
				s.credsHarnessNote = ""
			}
		}
		win := s.win
		s.mu.Unlock()
		if win != nil {
			win.Invalidate()
		}
	}()
}
