package projects

import (
	"regexp"
	"strings"
)

var generatedMemberUUID = regexp.MustCompile(`(?i)^member-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// The original hides only generated identifiers. A deliberate name such as
// member-frontend remains a visible name rather than becoming "Team member".
func generatedProjectMemberName(name, sessionID string) bool {
	name = strings.TrimSpace(name)
	return name == "" || name == "member-"+strings.TrimPrefix(sessionID, "session-") || generatedMemberUUID.MatchString(name)
}
