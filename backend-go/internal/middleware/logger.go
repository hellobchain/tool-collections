package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

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

		if method == "GET" {
			if query != "" {
				log.Printf("[%s] %s?%s", method, path, query)
			} else {
				log.Printf("[%s] %s", method, path)
			}
			c.Next()
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("[%s] %s - read body error: %v", method, path, err)
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if len(bodyBytes) > 0 {
			masked := maskSensitive(bodyBytes)
			log.Printf("[%s] %s body: %s", method, path, string(masked))
		} else {
			log.Printf("[%s] %s", method, path)
		}

		c.Next()
	}
}
