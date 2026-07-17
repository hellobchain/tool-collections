package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	CodeSuccess         = 0
	CodeError           = 1000
	CodeInvalidParams   = 400
	CodeUnauthorized    = 401
	CodeForbidden       = 402
	CodeNotFound        = 404
	CodeServerError     = 500
	CodeDBError         = 501
	CodeCacheError      = 502
	CodeTooManyRequests = 503
)

var codeMessages = map[int]string{
	CodeSuccess:         "success",
	CodeError:           "error",
	CodeInvalidParams:   "invalid parameters",
	CodeUnauthorized:    "unauthorized",
	CodeForbidden:       "forbidden",
	CodeNotFound:        "not found",
	CodeServerError:     "internal server error",
	CodeDBError:         "database error",
	CodeCacheError:      "cache error",
	CodeTooManyRequests: "too many requests",
}

type Response struct {
	Code      int         `json:"code"`
	Msg       string      `json:"msg"`
	Data      interface{} `json:"data,omitempty"`
	Errors    interface{} `json:"errors,omitempty"`
	Meta      interface{} `json:"meta,omitempty"`
	TraceID   string      `json:"trace_id,omitempty"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

type ResponseBuilder struct {
	code    int
	message string
	data    interface{}
	errors  interface{}
	meta    interface{}
	traceID string
}

func NewBuilder() *ResponseBuilder {
	return &ResponseBuilder{code: CodeSuccess}
}

func (rb *ResponseBuilder) WithCode(code int) *ResponseBuilder {
	rb.code = code
	if msg, ok := codeMessages[code]; ok {
		rb.message = msg
	}
	return rb
}

func (rb *ResponseBuilder) WithMessage(msg string) *ResponseBuilder {
	rb.message = msg
	return rb
}

func (rb *ResponseBuilder) WithData(data interface{}) *ResponseBuilder {
	rb.data = data
	return rb
}

func (rb *ResponseBuilder) WithErrors(errors interface{}) *ResponseBuilder {
	rb.errors = errors
	return rb
}

func (rb *ResponseBuilder) WithMeta(meta interface{}) *ResponseBuilder {
	rb.meta = meta
	return rb
}

func (rb *ResponseBuilder) WithTraceID(traceID string) *ResponseBuilder {
	rb.traceID = traceID
	return rb
}

func (rb *ResponseBuilder) Build() *Response {
	return &Response{
		Code:      rb.code,
		Msg:       rb.message,
		Data:      rb.data,
		Errors:    rb.errors,
		Meta:      rb.meta,
		TraceID:   rb.traceID,
		Timestamp: time.Now().Unix(),
	}
}

func (rb *ResponseBuilder) Send(c *gin.Context) {
	c.JSON(http.StatusOK, rb.Build())
}

func Success(c *gin.Context, data interface{}) {
	NewBuilder().WithCode(CodeSuccess).WithData(data).Send(c)
}

func SuccessWithMsg(c *gin.Context, data interface{}, msg string) {
	NewBuilder().WithCode(CodeSuccess).WithMessage(msg).WithData(data).Send(c)
}

func ErrorWithMsg(c *gin.Context, code int, msg string) {
	NewBuilder().WithCode(code).WithMessage(msg).Send(c)
}

type PageData struct {
	List       interface{} `json:"list"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

func SuccessPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 0 {
		totalPages = 0
	}
	data := PageData{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
	NewBuilder().WithCode(CodeSuccess).WithData(data).Send(c)
}

func SuccessList(c *gin.Context, list interface{}, total int64) {
	NewBuilder().WithCode(CodeSuccess).WithData(gin.H{"list": list, "total": total}).Send(c)
}
