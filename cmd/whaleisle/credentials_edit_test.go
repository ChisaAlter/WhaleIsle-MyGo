package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/credentials"
)

func TestCredentialsFailedLoadCannotBecomeEmptySavableDraft(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, credentials.FileName), []byte(`broken`), 0600); err != nil {
		t.Fatal(err)
	}
	s := &mainState{credsSeedGen: 1, credsSeedLoading: true}
	s.loadCredentialsDraftAsync(context.Background(), 1, dir)
	if s.credsDraft.seeded || !s.credsDraft.loadFailed || s.credentialsCanSave() {
		t.Fatal("failed load became savable")
	}
	if err := os.WriteFile(filepath.Join(dir, credentials.FileName), []byte(`{"apiKey":"restored","githubToken":"keep"}`), 0600); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.saveCredentials()
	s.mu.Unlock()
	got, err := credentials.Load(dir)
	if err != nil || got.APIKey != "restored" || got.GithubToken != "keep" {
		t.Fatal("failed draft overwrote restored file", err)
	}
	s.credsDraft.loadFailed = false
	s.credsSeedGen = 2
	s.loadCredentialsDraftAsync(context.Background(), 2, dir)
	if !s.credentialsLoaded() || s.credsDraft.apiKey != "restored" {
		t.Fatal("explicit retry did not load latest")
	}
	if wantSave := s.credsDraft.editor.WriteAccessError() == nil; s.credentialsCanSave() != wantSave {
		t.Fatal("explicit retry did not reflect the shared profile's write availability")
	}
}

func TestCredentialsCommitReconcilesWithoutClearingContinuedTyping(t *testing.T) {
	current := credentialsDraft{apiKey: "typed-later", githubToken: "old"}
	sent := credentials.Fields{APIKey: "submitted", GithubToken: "old"}
	fresh := credentialsDraft{apiKey: "submitted", githubToken: "peer-new"}
	keepCredentialEdits(&current, sent, fresh)
	if current.apiKey != "typed-later" || current.githubToken != "peer-new" {
		t.Fatal("continued typing or independent update lost")
	}
}

func TestCredentialsUnavailableSharedWriteKeepsLoadedDraft(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, credentials.FileName)
	if err := os.WriteFile(file, []byte(`{"apiKey":"original"}`), 0600); err != nil {
		t.Fatal(err)
	}
	editor, err := credentials.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{credsDraft: credentialDraftFromEditor(editor)}
	s.credsDraft.apiKey = "unsaved-edit"
	s.credsDraft.writeAccessErr = credentials.ErrSharedProfileWriteUnavailable.Error()
	if !s.credentialsLoaded() || s.credentialsCanSave() {
		t.Fatal("write restriction changed read availability or enabled saving")
	}
	s.mu.Lock()
	s.saveCredentials()
	s.mu.Unlock()
	if s.credsDraft.apiKey != "unsaved-edit" || s.credsDraft.saving || s.credsSaveErr != s.credsDraft.writeAccessErr {
		t.Fatal("blocked save lost its draft or status")
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != `{"apiKey":"original"}` {
		t.Fatal("blocked editor changed the persisted credentials", err)
	}
}
