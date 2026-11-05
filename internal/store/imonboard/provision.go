package imonboard

import "strings"

// Verification codes are the digits displayed by Weixin for this attempt.
func ValidVerificationCode(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 4 || len(value) > 8 {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

func (s *Store) ClearCredentialDraft(scope uint64) {
	s.mu.Lock()
	if s.view.Scope == scope && !s.view.Busy {
		s.view.Draft = Draft{}
		s.view.Error = ""
	}
	s.mu.Unlock()
	s.fire()
}

// Closing a terminal presentation is not another provisioning mutation.
func (s *Store) CloseProvision(scope uint64) {
	s.mu.Lock()
	if s.view.Scope == scope && !s.view.Busy && (s.view.Provision.Status == "failed" || s.view.Provision.Status == "expired" || s.view.Provision.Status == "cancelled") {
		s.view.Provision = Provision{}
		s.view.Draft.VerifyCode = ""
		s.view.Error = ""
	}
	s.mu.Unlock()
	s.fire()
}

func normalizeAccessEntries(channel, text string) ([]string, string) {
	values := []string{}
	seen := map[string]bool{}
	for _, entry := range strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' || r == ',' }) {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		minimum, maximum := 1, 16
		message := "Telegram User ID 必须是 1–16 位正整数，每行一个。"
		if channel == "whatsapp" {
			entry = strings.TrimPrefix(entry, "+")
			minimum, maximum = 5, 15
			message = "电话号码必须是包含国家或地区码的 5–15 位数字，每行一个。"
		}
		if len(entry) < minimum || len(entry) > maximum || entry[0] == '0' {
			return nil, message
		}
		for _, digit := range entry {
			if digit < '0' || digit > '9' {
				return nil, message
			}
		}
		if !seen[entry] {
			values = append(values, entry)
			seen[entry] = true
		}
	}
	return values, ""
}
