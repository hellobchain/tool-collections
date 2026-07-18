package ginlog

import (
	"bytes"
	"io"
	"strings"
)

// extractRequestParams 提取请求参数
func extractRequestParams(ctx *RequestContext) string {
	if ctx.Method == "GET" {
		return formatGetParams(ctx.Query)
	}
	return extractBodyParams(ctx)
}

// formatGetParams 格式化GET请求参数
func formatGetParams(query string) string {
	if query != "" {
		return "?" + query
	}
	return ""
}

// extractBodyParams 提取请求体参数
func extractBodyParams(ctx *RequestContext) string {
	// 判断是否为文件上传
	if isMultipartRequest(ctx.ContentType) {
		return " (file upload, body omitted)"
	}

	return readAndMaskBody(ctx)
}

// isMultipartRequest 判断是否为multipart请求
func isMultipartRequest(contentType string) bool {
	return strings.HasPrefix(contentType, "multipart/form-data")
}

// readAndMaskBody 读取并脱敏请求体
func readAndMaskBody(ctx *RequestContext) string {
	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		slog.Errorf("[%s] %s - read body error: %v",
			ctx.Method, ctx.Path, err)
		return "read body error"
	}

	// 恢复请求体供后续使用
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if len(bodyBytes) == 0 {
		return "no body"
	}

	// 脱敏处理
	masked := maskSensitive(bodyBytes)
	return " - body: " + string(masked)
}
