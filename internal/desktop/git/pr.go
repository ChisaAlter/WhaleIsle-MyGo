package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// MsgMissingGH is returned when the GitHub CLI is not on PATH.
const MsgMissingGH = "未找到 GitHub CLI (gh)"

// PRStatus is a lightweight pull-request summary from `gh` when available.
type PRStatus struct {
	OK               bool
	Summary          string
	URL              string
	Host, Repository string
	Err              string
}

// PRInfo is one pull request from gh JSON (status / list / view).
type PRInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	State  string `json:"state"`
}

// PRReport aggregates `gh pr status` plus a short `gh pr list`.
type PRReport struct {
	OK          bool
	MissingGH   bool
	Current     *PRInfo
	CreatedBy   []PRInfo
	NeedsReview []PRInfo
	Recent      []PRInfo
	Err         string
}

// DefaultPRTimeout bounds gh pr view / create / status / list / checkout.
const DefaultPRTimeout = 20 * time.Second

// DefaultPRListLimit is used by PRStatusReport for `gh pr list --limit`.
const DefaultPRListLimit = 5

// PathLooker locates an executable (overridden in tests).
type PathLooker func(file string) (string, error)

// ViewPR runs `gh pr view --json number,title,url,state` in root when gh is on PATH.
func ViewPR(ctx context.Context, root string) PRStatus {
	return viewPR(ctx, root, nil, combinedExecRunner)
}

func viewPR(ctx context.Context, root string, look PathLooker, run combinedRunner) PRStatus {
	if err := requireGH(look); err != "" {
		return PRStatus{Err: err}
	}
	ctx, cancel := ensurePRContext(ctx)
	defer cancel()
	repo, host, err := resolvePRRepository(ctx, root, run)
	if err != nil {
		return PRStatus{Err: err.Error()}
	}
	// Let gh resolve the current branch, including its fork/upstream metadata.
	// --repo requires an explicit PR selector and disables that lookup.
	out, err := run(ctx, root, "gh", "pr", "view", "--json", "number,title,url,state")
	if err != nil {
		text := strings.TrimSpace(string(out))
		if text == "" {
			text = err.Error()
		}
		return PRStatus{Err: text}
	}
	info, err := parsePRInfoJSON(out)
	if err != nil || !ValidPullRequestURLForHost(info.URL, host, repo.NameWithOwner) {
		return PRStatus{Err: "GitHub returned an unexpected PR URL"}
	}
	return PRStatus{OK: true, Summary: FormatPRInfo(info), URL: info.URL, Host: host, Repository: repo.NameWithOwner}

}

// CreatePRDraft runs `gh pr create --draft --fill` when gh is available.
func CreatePRDraft(ctx context.Context, root string) PRStatus {
	return createPRDraft(ctx, root, nil, combinedExecRunner)
}

func createPRDraft(ctx context.Context, root string, look PathLooker, run combinedRunner) PRStatus {
	if err := requireGH(look); err != "" {
		return PRStatus{Err: err}
	}
	ctx, cancel := ensurePRContext(ctx)
	defer cancel()
	out, err := run(ctx, root, "gh", "pr", "create", "--draft", "--fill")
	text := strings.TrimSpace(string(out))
	if err != nil {
		msg := text
		if msg == "" {
			msg = err.Error()
		}
		return PRStatus{Err: fmt.Sprintf("创建 PR 失败：%s", msg)}
	}
	return PRStatus{OK: true, Summary: text, URL: firstHTTPURL(text)}
}

// CheckoutPR runs `gh pr checkout <number>` when gh is available.
func CheckoutPR(ctx context.Context, root string, number int) PRStatus {
	return checkoutPR(ctx, root, number, nil, combinedExecRunner)
}

func checkoutPR(ctx context.Context, root string, number int, look PathLooker, run combinedRunner) PRStatus {
	if number <= 0 {
		return PRStatus{Err: "invalid PR number"}
	}
	if err := requireGH(look); err != "" {
		return PRStatus{Err: err}
	}
	ctx, cancel := ensurePRContext(ctx)
	defer cancel()
	out, err := run(ctx, root, "gh", "pr", "checkout", fmt.Sprintf("%d", number))
	text := strings.TrimSpace(string(out))
	if err != nil {
		msg := text
		if msg == "" {
			msg = err.Error()
		}
		return PRStatus{Err: fmt.Sprintf("checkout PR 失败：%s", msg)}
	}
	if text == "" {
		text = fmt.Sprintf("checked out #%d", number)
	}
	return PRStatus{OK: true, Summary: text}
}

// PRStatusReport runs `gh pr status` and `gh pr list --limit 5` (JSON).
func PRStatusReport(ctx context.Context, root string) PRReport {
	return prStatusReport(ctx, root, DefaultPRListLimit, nil, combinedExecRunner)
}

func prStatusReport(ctx context.Context, root string, limit int, look PathLooker, run combinedRunner) PRReport {
	if err := requireGH(look); err != "" {
		return PRReport{MissingGH: true, Err: err}
	}
	if limit <= 0 {
		limit = DefaultPRListLimit
	}
	ctx, cancel := ensurePRContext(ctx)
	defer cancel()

	statusOut, err := run(ctx, root, "gh", "pr", "status", "--json", "number,title,url,state")
	if err != nil {
		msg := strings.TrimSpace(string(statusOut))
		if msg == "" {
			msg = err.Error()
		}
		return PRReport{Err: msg}
	}
	current, createdBy, needsReview, perr := ParsePRStatusJSON(statusOut)
	if perr != nil {
		return PRReport{Err: perr.Error()}
	}

	rep := PRReport{
		OK:          true,
		Current:     current,
		CreatedBy:   createdBy,
		NeedsReview: needsReview,
	}

	listOut, lerr := run(ctx, root, "gh", "pr", "list", "--limit", fmt.Sprintf("%d", limit), "--json", "number,title,url,state")
	if lerr != nil {
		// Status succeeded; keep it and surface list error softly.
		msg := strings.TrimSpace(string(listOut))
		if msg == "" {
			msg = lerr.Error()
		}
		rep.Err = msg
		return rep
	}
	recent, lperr := ParsePRListJSON(listOut)
	if lperr != nil {
		rep.Err = lperr.Error()
		return rep
	}
	rep.Recent = recent
	return rep
}

// ParsePRStatusJSON parses `gh pr status --json number,title,url,state` output.
func ParsePRStatusJSON(data []byte) (current *PRInfo, createdBy, needsReview []PRInfo, err error) {
	var payload struct {
		CurrentBranch json.RawMessage `json:"currentBranch"`
		CreatedBy     []PRInfo        `json:"createdBy"`
		NeedsReview   []PRInfo        `json:"needsReview"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, nil, nil, err
	}
	createdBy = payload.CreatedBy
	if createdBy == nil {
		createdBy = []PRInfo{}
	}
	needsReview = payload.NeedsReview
	if needsReview == nil {
		needsReview = []PRInfo{}
	}
	raw := bytesTrimSpace(payload.CurrentBranch)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, createdBy, needsReview, nil
	}
	info, err := parsePRInfoJSON(raw)
	if err != nil {
		return nil, createdBy, needsReview, err
	}
	if info.Number == 0 && info.Title == "" && info.URL == "" {
		return nil, createdBy, needsReview, nil
	}
	return &info, createdBy, needsReview, nil
}

// ParsePRListJSON parses `gh pr list --json number,title,url,state` output.
func ParsePRListJSON(data []byte) ([]PRInfo, error) {
	var list []PRInfo
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	if list == nil {
		list = []PRInfo{}
	}
	return list, nil
}

// FormatPRInfo returns a compact label like "#12 Fix login [OPEN]".
func FormatPRInfo(p PRInfo) string {
	var b strings.Builder
	if p.Number > 0 {
		fmt.Fprintf(&b, "#%d", p.Number)
	}
	if t := strings.TrimSpace(p.Title); t != "" {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(t)
	}
	if s := strings.TrimSpace(p.State); s != "" {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "[%s]", s)
	}
	return b.String()
}

func parsePRInfoJSON(data []byte) (PRInfo, error) {
	var info PRInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return PRInfo{}, err
	}
	return info, nil
}

func requireGH(look PathLooker) string {
	if look == nil {
		look = exec.LookPath
	}
	if _, err := look("gh"); err != nil {
		return MsgMissingGH
	}
	return ""
}

func ensurePRContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx != nil {
		return ctx, func() {}
	}
	return context.WithTimeout(context.Background(), DefaultPRTimeout)
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}

func firstHTTPURL(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			return line
		}
	}
	return ""
}
