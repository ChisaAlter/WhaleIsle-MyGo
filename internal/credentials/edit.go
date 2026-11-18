package credentials

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// Fields is the desktop credential editor's complete write surface. Device
// registrations and encrypted namespaces remain with their existing owners.
type Fields struct {
	APIKey           string
	BaseURL          string
	GithubToken      string
	RemoteToken      string
	RemoteRelayToken string
}

func fieldsOf(c Credentials) Fields {
	return Fields{c.APIKey, c.BaseURL, c.GithubToken, c.RemoteToken, c.RemoteRelayToken}
}

// Fields returns the values at the editor's last successful load or commit.
func (s *Store) Fields() Fields { return fieldsOf(s.Get()) }

// CommitFields merges only edits relative to this Store's loaded values.
// Unedited fields and namespaces come from the latest file. A conflicting
// change to an edited field leaves both the file and the editor unchanged.
// The final baseline check detects non-cooperating writers observed before
// rename; .credentials.lock serializes participating MyGo writers only.
func (s *Store) CommitFields(edited Fields) (Credentials, error) {
	s.mu.RLock()
	base := fieldsOf(s.creds)
	s.mu.RUnlock()
	var committed Credentials
	var rawAfter map[string]json.RawMessage
	var baselineAfter fileBaseline
	defer func() { clearRawMap(rawAfter) }()
	err := withProfileDirLock(s.dir, true, s.shared, func() error {
		raw, create, baseline, err := readExistingRawForSave(s.dir)
		if err != nil {
			return err
		}
		defer clearRawMap(raw)
		encoded, err := json.Marshal(raw)
		if err != nil {
			return err
		}
		defer clear(encoded)
		if err := json.Unmarshal(encoded, &committed); err != nil {
			return err
		}
		changed := false
		for _, field := range []struct {
			name         string
			base, edited string
			current      *string
		}{
			{"apiKey", base.APIKey, edited.APIKey, &committed.APIKey},
			{"baseUrl", base.BaseURL, edited.BaseURL, &committed.BaseURL},
			{"githubToken", base.GithubToken, edited.GithubToken, &committed.GithubToken},
			{"remoteToken", base.RemoteToken, edited.RemoteToken, &committed.RemoteToken},
			{"remoteRelayToken", base.RemoteRelayToken, edited.RemoteRelayToken, &committed.RemoteRelayToken},
		} {
			if field.edited == field.base {
				continue
			}
			value := strings.TrimSpace(field.edited)
			if *field.current != field.base && *field.current != value {
				return fmt.Errorf("%w: %s", ErrConflict, field.name)
			}
			if value != *field.current {
				*field.current = value
				raw[field.name], _ = json.Marshal(value)
				changed = true
			}
		}
		if changed {
			if err := sealAndWriteRaw(s.dir, raw, create, guardedRename(filepath.Join(s.dir, FileName), baseline)); err != nil {
				return err
			}
			baseline, err = readFileBaseline(s.path)
			if err != nil {
				return err
			}
		}
		rawAfter, baselineAfter = cloneRawMap(raw), baseline
		return nil
	})
	if err != nil {
		return Credentials{}, err
	}
	s.mu.Lock()
	clearRawMap(s.raw)
	s.creds, s.raw, s.baseline = cloneCredentials(committed), cloneRawMap(rawAfter), baselineAfter
	s.mu.Unlock()
	return cloneCredentials(committed), nil
}
