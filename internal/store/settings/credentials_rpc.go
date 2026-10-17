package settings

import (
	"context"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// Common credential refs used by Electron settings credentials panel.
const (
	CredRefAPIKey      = "apiKey"
	CredRefGithubToken = "githubToken"
)

// DescribeCredentials calls credentials/describe for the given refs.
func DescribeCredentials(ctx context.Context, client *dsh.Client, refs []string) (api.DescribeResult, error) {
	var out api.DescribeResult
	if client == nil {
		return out, nil
	}
	err := client.Call(ctx, "credentials/describe", map[string]any{"refs": refs}, &out)
	return out, err
}

// SetCredential calls credentials/set.
func SetCredential(ctx context.Context, client *dsh.Client, ref, value string) error {
	if client == nil || strings.TrimSpace(ref) == "" {
		return nil
	}
	var out api.SetResult
	return client.Call(ctx, "credentials/set", map[string]any{
		"ref":   ref,
		"value": value,
	}, &out)
}

// UnsetCredential calls credentials/unset.
func UnsetCredential(ctx context.Context, client *dsh.Client, ref string) error {
	if client == nil || strings.TrimSpace(ref) == "" {
		return nil
	}
	var out api.UnsetResult
	return client.Call(ctx, "credentials/unset", map[string]any{"ref": ref}, &out)
}

// SyncAPIKeyToHarness mirrors local API key into harness credentials store.
func SyncAPIKeyToHarness(ctx context.Context, client *dsh.Client, apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if client == nil {
		return nil
	}
	if apiKey == "" {
		return UnsetCredential(ctx, client, CredRefAPIKey)
	}
	return SetCredential(ctx, client, CredRefAPIKey, apiKey)
}
