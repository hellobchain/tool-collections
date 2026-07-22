package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
	"github.com/hellobchain/wswlog/wlogging"
)

var jsonlSlog = wlogging.MustGetLoggerWithoutName()

const (
	maxLimit = 10000
)

// StreamJSONL godoc
// @Summary Stream JSONL file contents
// @Description Stream records from a JSONL file with offset/limit support, using SSE for large files.
// @Description Supports any JSON structure in the JSONL file. Optionally accepts a `schema` parameter
// @Description (URL-encoded JSON object) to validate field presence.
// @Tags JSONL
// @Param file query string true "Path to the JSONL file"
// @Param offset query int false "Line offset to start from (0-based)" default(0)
// @Param limit query int false "Maximum number of records to return" default(1000) maximum(10000)
// @Param schema query string false "URL-encoded JSON object defining expected fields for validation"
// @Success 200 {object} services.GenericRecord "SSE stream of records"
// @Failure 400 {object} utils.Response "Invalid parameters"
// @Failure 404 {object} utils.Response "File not found"
// @Failure 500 {object} utils.Response "Internal server error"
// @Router /api/v1/data/stream [get]
func StreamJSONL(c *gin.Context) {
	filePath := c.Query("file")
	if filePath == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "file parameter is required")
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	limitStr := c.DefaultQuery("limit", "1000")
	schemaStr := c.Query("schema")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "offset must be a non-negative integer")
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "limit must be a non-negative integer")
		return
	}
	if limit > maxLimit {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, fmt.Sprintf("limit must not exceed %d", maxLimit))
		return
	}

	validPath, err := services.ValidateFilePath(config.AppConfig.JSONLAllowedDir, filePath)
	if err != nil {
		jsonlSlog.Warnf("JSONL path validation failed: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, err.Error())
		return
	}

	if !fileExists(validPath) {
		utils.ErrorWithMsg(c, utils.CodeNotFound, fmt.Sprintf("file not found: %s", filePath))
		return
	}

	ctx := c.Request.Context()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ch := make(chan services.StreamResult, 64)

	if schemaStr != "" {
		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
			utils.ErrorWithMsg(c, utils.CodeInvalidParams, "schema must be a valid JSON object")
			return
		}
		go services.ReadJSONLStreamWithSchema(ctx, validPath, offset, limit, schema, ch)
	} else {
		go services.ReadJSONLStream(ctx, validPath, offset, limit, ch)
	}

	flusher, flushOk := c.Writer.(http.Flusher)
	if !flushOk {
		utils.ErrorWithMsg(c, utils.CodeServerError, "streaming not supported")
		return
	}

	stats := services.StreamStats{}
	for result := range ch {
		if result.Err != nil {
			stats.ErrorLines++
			if result.Record == nil && result.Line == 0 {
				jsonlSlog.Errorf("JSONL stream fatal error: %v", result.Err)
				break
			}
			errMsg := fmt.Sprintf(`{"line":%d,"error":"%s"}`, result.Line, result.Err.Error())
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", errMsg)
			flusher.Flush()
			continue
		}

		if result.Record != nil {
			stats.ReadLines++
			stats.ReturnLines++
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", string(result.Record.Raw))
			flusher.Flush()
		}
	}

	doneMsg := fmt.Sprintf(`{"code":0,"msg":"done","stats":{"read_lines":%d,"error_lines":%d,"return_lines":%d}}`,
		stats.ReadLines, stats.ErrorLines, stats.ReturnLines)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", doneMsg)
	flusher.Flush()

	jsonlSlog.Infof("JSONL stream finished: path=%s offset=%d limit=%d read=%d errors=%d returned=%d",
		validPath, offset, limit, stats.ReadLines, stats.ErrorLines, stats.ReturnLines)
}

func fileExists(path string) bool {
	info, err := services.FileExists(path)
	return err == nil && !info.IsDir()
}
