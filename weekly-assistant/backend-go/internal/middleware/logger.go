package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hellobchain/wswlog/wlogging"
)

var slog = wlogging.MustGetLoggerWithoutName()
var sensitiveKeys = []string{"password", "token", "refresh_token", "old_password", "new_password"}

func maskSensitive(body []byte) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body
	}
	for k := range data {
		for _, key := range sensitiveKeys {
			if strings.EqualFold(k, key) {
				data[k] = "***"
			}
		}
	}
	masked, _ := json.Marshal(data)
	return masked
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		contentType := c.Request.Header.Get("Content-Type")

		// 记录请求路径
		if method == "GET" {
			if query != "" {
				slog.Infof("[%s] %s?%s", method, path, query)
			} else {
				slog.Infof("[%s] %s", method, path)
			}
			c.Next()
			return
		}

		// 判断是否为文件上传（multipart/form-data）
		isMultipart := strings.HasPrefix(contentType, "multipart/form-data")

		// 如果是文件上传，只记录路径，不读取body
		if isMultipart {
			slog.Infof("[%s] %s (file upload, body omitted)", method, path)
			c.Next()
			return
		}

		// 非文件上传，读取并记录body
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			slog.Errorf("[%s] %s - read body error: %v", method, path, err)
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if len(bodyBytes) > 0 {
			masked := maskSensitive(bodyBytes)
			slog.Infof("[%s] %s body: %s", method, path, string(masked))
		} else {
			slog.Infof("[%s] %s", method, path)
		}

		c.Next()
	}
}
