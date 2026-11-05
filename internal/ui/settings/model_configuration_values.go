package settings

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func configurationObject(raw string) map[string]any {
	var v map[string]any
	_ = json.Unmarshal([]byte(raw), &v)
	if v == nil {
		v = map[string]any{}
	}
	return v
}
func configurationRows(value any) []any {
	v, _ := value.([]any)
	if v == nil {
		return []any{}
	}
	return v
}
func configurationText(value any) string {
	if value == nil {
		return ""
	}
	if n, ok := value.(float64); ok && !math.IsNaN(n) && !math.IsInf(n, 0) && math.Trunc(n) == n {
		return strconv.FormatInt(int64(n), 10)
	}
	return fmt.Sprint(value)
}

func configurationFormatCapacity(value any) string {
	if value == nil {
		return ""
	}
	if n, ok := value.(float64); ok && !math.IsNaN(n) && !math.IsInf(n, 0) && n > 0 && math.Trunc(n) == n {
		i := int64(n)
		if i%1_000_000 == 0 {
			return fmt.Sprintf("%dM", i/1_000_000)
		}
		if i%1_000 == 0 {
			return fmt.Sprintf("%dK", i/1_000)
		}
		return strconv.FormatInt(i, 10)
	}
	return configurationText(value)
}
func configurationCapacity(value string) (any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	scale := float64(1)
	upper := strings.ToUpper(value)
	if strings.HasSuffix(upper, "K") {
		scale = 1000
		value = value[:len(value)-1]
	} else if strings.HasSuffix(upper, "M") {
		scale = 1000000
		value = value[:len(value)-1]
	}
	n, err := strconv.ParseFloat(value, 64)
	n *= scale
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 || math.Trunc(n) != n {
		return nil, fmt.Errorf("Use a positive whole token count, optionally K or M")
	}
	return n, nil
}
func configurationModelError(rows []any) string {
	seen := map[string]bool{}
	for i, raw := range rows {
		row, _ := raw.(map[string]any)
		id := strings.TrimSpace(configurationText(row["id"]))
		if id == "" || seen[id] {
			return fmt.Sprintf("Model %d: enter a unique model ID", i+1)
		}
		seen[id] = true
		for _, key := range []string{"input", "inputModalities"} {
			if value, exists := row[key]; exists {
				values := configurationRows(value)
				if len(values) == 0 {
					return fmt.Sprintf("Model %d: %s", i+1, providerCopy("至少保留一种输入类型", "Keep at least one input type"))
				}
				if key == "inputModalities" {
					image := false
					for _, v := range values {
						image = image || v == "image"
					}
					if !image && (row["imagePixelBudget"] != nil || row["imageMaxBytes"] != nil) {
						return fmt.Sprintf("Model %d: %s", i+1, providerCopy("纯文本模型不能设置图片限制", "Text-only models cannot set image limits"))
					}
				}
			}
		}
		if efforts, has := row["reasoningEfforts"].(map[string]any); has {
			nonOff := 0
			emptyValue := false
			for level, val := range efforts {
				if level != "off" {
					nonOff++
					if strings.TrimSpace(configurationText(val)) == "" {
						emptyValue = true
					}
				}
			}
			if nonOff == 0 || emptyValue {
				return fmt.Sprintf("Model %d: %s", i+1, providerCopy("至少保留一个非关闭的推理思考等级", "Keep at least one non-off reasoning level"))
			}
		}
		for _, key := range []string{"contextWindow", "maxTokens"} {
			if value, exists := row[key]; exists {
				if _, err := configurationCapacity(configurationText(value)); err != nil {
					return fmt.Sprintf("Model %d: %s: %s", i+1, key, err)
				}
			}
		}
	}
	return ""
}
func configurationCreateError(route, endpoint, protocol string, rows []any, providers map[string]any) string {
	if !regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`).MatchString(route) {
		return "Route ID: use lowercase letters, digits and hyphens; start with a letter"
	}
	if _, exists := providers[route]; exists {
		return "This provider route already exists"
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "Enter an HTTP(S) endpoint"
	}
	if protocol == "" {
		return "Choose a Host-supported protocol"
	}
	if len(rows) == 0 {
		return "Add at least one model"
	}
	return configurationModelError(rows)
}
