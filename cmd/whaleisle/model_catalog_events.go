package main

import (
	"context"

	"whaleisle/internal/store/session"
)

// handleModelCatalogEvent dispatches Host 0.2.2 $events that affect the model
// catalog, matching the official ui-model-selection subscription list.
func handleModelCatalogEvent(ctx context.Context, sessions *session.Store, event string) {
	if sessions == nil || ctx.Err() != nil {
		return
	}
	switch event {
	case "credentials/record-updated":
		sessions.InitializeDefaultModel(ctx)
	case "llm/adapters-updated", "llm/models-updated", "settings/document-updated", "credentials/reference-updated":
		sessions.RefreshModelCatalog(ctx)
	}
}

// handleAccountCredentialsChanged refreshes the model catalog after account/watch
// reports credential changes, seeding the default model on credential-stored.
func handleAccountCredentialsChanged(ctx context.Context, sessions *session.Store, status string) {
	if sessions == nil || ctx.Err() != nil {
		return
	}
	if status == "credential-stored" {
		sessions.InitializeDefaultModel(ctx)
		return
	}
	sessions.RefreshModelCatalog(ctx)
}
