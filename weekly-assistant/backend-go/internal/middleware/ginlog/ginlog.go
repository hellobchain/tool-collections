package ginlog

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hellobchain/wswlog/wlogging"
)

var slog = wlogging.MustGetLoggerWithoutName()

// Logger 请求日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 构建请求上下文
		reqCtx := &RequestContext{
			Context:     c,
			StartTime:   start,
			Path:        c.Request.URL.Path,
			Query:       c.Request.URL.RawQuery,
			Method:      c.Request.Method,
			ContentType: c.Request.Header.Get("Content-Type"),
			ClientIP:    c.ClientIP(),
		}

		// 提取请求参数
		reqCtx.Params = extractRequestParams(reqCtx)

		// 处理请求
		c.Next()

		// 记录日志
		logRequest(reqCtx)
	}
}

// RequestContext 请求上下文
type RequestContext struct {
	*gin.Context
	StartTime   time.Time
	Path        string
	Query       string
	Method      string
	ContentType string
	ClientIP    string
	Params      string
}

// logRequest 记录请求日志
func logRequest(ctx *RequestContext) {
	latency := time.Since(ctx.StartTime)
	statusCode := ctx.Writer.Status()

	slog.Infof("| %3d | %13v | %15s | %s | %s | %s |",
		statusCode,
		latency,
		ctx.ClientIP,
		ctx.Method,
		ctx.Path,
		ctx.Params,
	)
}
