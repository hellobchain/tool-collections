package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

const (
	maxLimit = 10000
)

/*
 * @Description: Stream JSONL file
 */
func StreamJSONL(c *gin.Context) {
	var req models.JsonlReaderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}
	filePath := req.FilePath
	if filePath == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "file parameter is required")
		return
	}

	offset := req.Offset
	limit := req.Limit
	schemaStr := req.Schema
	if offset < 0 {
		slog.Warnf("JSONL offset must be positive: %d default to 0", offset)
		offset = 0
	}

	if limit < 0 {
		slog.Warnf("JSONL limit must be positive: %d default to 1000", limit)
		limit = 1000
	}
	if limit > maxLimit {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, fmt.Sprintf("limit must not exceed %d", maxLimit))
		return
	}

	validPath, err := services.ValidateFilePath(config.AppConfig.JSONLAllowedDir, filePath)
	if err != nil {
		slog.Warnf("JSONL path validation failed: %v", err)
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
		recordStr, err := result.Record.MarshalJSON()
		if err != nil {
			slog.Warnf("JSONL stream marshal error: %v", err)
		}
		slog.Infof("JSONL stream result: path=%s line=%d record=%v", validPath, result.Line, string(recordStr))
		if result.Err != nil {
			stats.ErrorLines++
			if result.Record == nil && result.Line == 0 {
				slog.Errorf("JSONL stream fatal error: %v", result.Err)
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
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", "[DONE]")
	flusher.Flush()

	slog.Infof("JSONL stream finished: path=%s offset=%d limit=%d read=%d errors=%d returned=%d",
		validPath, offset, limit, stats.ReadLines, stats.ErrorLines, stats.ReturnLines)
}

func fileExists(path string) bool {
	info, err := services.FileExists(path)
	return err == nil && !info.IsDir()
}
