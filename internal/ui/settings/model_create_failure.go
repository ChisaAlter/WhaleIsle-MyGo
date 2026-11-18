package settings

import (
	"strings"

	store "whaleisle/internal/store/settings"
)

// providerCreateFailure is what a failed custom-provider creation says: one
// sentence in the reader's language that names the next step, plus the Host's
// own words for the details block. A Host refusal or a raw transport message
// never stands in for the sentence, because it is written for a log.
func providerCreateFailure(snap store.ViewState, keyError string) (sentence string, details []string) {
	sentence = snap.Err
	if strings.TrimSpace(sentence) == "" {
		sentence = providerCopy("提供商没有创建成功，API 密钥也没有保存。草稿已保留，请再试一次；如果还是失败，请导出日志后联系我们。", "The provider was not created and its API key was not saved. Your edits are kept as a draft; try again, and if it keeps failing, export the logs and contact us.")
	}
	for _, line := range []string{snap.ErrDetail, keyError} {
		line = strings.TrimSpace(line)
		if line == "" || line == sentence {
			continue
		}
		duplicate := false
		for _, kept := range details {
			duplicate = duplicate || kept == line
		}
		if !duplicate {
			details = append(details, line)
		}
	}
	return sentence, details
}
