package git

import (
	"context"
	"errors"
	"whaleisle/internal/credentials"
	"whaleisle/internal/paths"
)

// GenerateChangeRequestText shares desktop generation without exporting credentials.
// Its caller supplies the already collected, authorized committed branch range.
func GenerateChangeRequestText(ctx context.Context, in PRGenerationInput) (GeneratedPRContent, error) {
	creds, err := credentials.Load(paths.UserDataDir())
	if err != nil {
		return GeneratedPRContent{}, errors.New("cannot read saved generation credentials; keep editing manually")
	}
	return generatePRText(ctx, in, creds.APIKey, creds.BaseURL, nil)
}
