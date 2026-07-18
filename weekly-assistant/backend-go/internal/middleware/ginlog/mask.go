package ginlog

import (
	"encoding/json"
	"regexp"
	"strings"
)

// 敏感字段列表
var sensitiveFields = map[string]bool{
	"password":      true,
	"pwd":           true,
	"passwd":        true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"secret":        true,
	"api_key":       true,
	"apikey":        true,
	"authorization": true,
	"auth":          true,
}

// maskSensitive 脱敏敏感数据
func maskSensitive(data []byte) []byte {
	// 尝试作为JSON解析
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		// JSON对象，递归脱敏
		maskedObj := maskJSONObject(obj)
		if result, err := json.Marshal(maskedObj); err == nil {
			return result
		}
	}

	// 非JSON或解析失败，尝试字符串替换
	return []byte(maskString(string(data)))
}

// maskJSONObject 递归脱敏JSON对象
func maskJSONObject(obj interface{}) interface{} {
	switch v := obj.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			if isSensitiveField(key) {
				result[key] = "***MASKED***"
			} else {
				result[key] = maskJSONObject(val)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = maskJSONObject(val)
		}
		return result
	default:
		return v
	}
}

// isSensitiveField 判断是否为敏感字段
func isSensitiveField(field string) bool {
	fieldLower := strings.ToLower(field)
	for pattern := range sensitiveFields {
		if strings.Contains(fieldLower, pattern) {
			return true
		}
	}
	return false
}

// maskString 对字符串进行脱敏
func maskString(data string) string {
	patterns := []struct {
		regex *regexp.Regexp
		repl  string
	}{
		// 密码类
		{regexp.MustCompile(`(?i)"(password|pwd|passwd)"\s*:\s*"[^"]*"`), `"$1":"***MASKED***"`},
		{regexp.MustCompile(`(?i)"(password|pwd|passwd)"\s*:\s*'[^']*'`), `"$1":"***MASKED***"`},
		// Token类
		{regexp.MustCompile(`(?i)"(token|access_token|refresh_token)"\s*:\s*"[^"]*"`), `"$1":"***MASKED***"`},
		// 其他敏感信息
		{regexp.MustCompile(`(?i)"(secret|api_key|apikey)"\s*:\s*"[^"]*"`), `"$1":"***MASKED***"`},
	}

	result := data
	for _, p := range patterns {
		result = p.regex.ReplaceAllString(result, p.repl)
	}
	return result
}
