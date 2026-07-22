package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func ConvertJSONLToDocx(c *gin.Context) {
	var req models.JsonlReaderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, err.Error())
		return
	}
	if req.FilePath == "" {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, "file_path is required")
		return
	}

	offset := req.Offset
	limit := req.Limit
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 1000
	}
	if limit > maxLimit {
		utils.ErrorWithMsg(c, utils.CodeInvalidParams, fmt.Sprintf("limit must not exceed %d", maxLimit))
		return
	}

	validPath, err := services.ValidateFilePath(config.AppConfig.JSONLAllowedDir, req.FilePath)
	if err != nil {
		slog.Warnf("JSONL path validation failed: %v", err)
		utils.ErrorWithMsg(c, utils.CodeNotFound, err.Error())
		return
	}

	if !fileExists(validPath) {
		utils.ErrorWithMsg(c, utils.CodeNotFound, fmt.Sprintf("file not found: %s", req.FilePath))
		return
	}

	ctx := c.Request.Context()

	c.Header("Content-Type", "application/x-ndjson")
	c.Header("X-Content-Type-Options", "nosniff")

	ch := make(chan services.StreamResult, 64)
	go services.ReadJSONLStream(ctx, validPath, offset, limit, ch)

	flusher, flushOk := c.Writer.(http.Flusher)
	if !flushOk {
		utils.ErrorWithMsg(c, utils.CodeServerError, "streaming not supported")
		return
	}

	stats := services.StreamStats{}
	var mu sync.Mutex
	converted := make([]map[string]interface{}, 0)

	for result := range ch {
		if result.Err != nil {
			stats.ErrorLines++
			if result.Record == nil && result.Line == 0 {
				slog.Errorf("JSONL convert fatal error: %v", result.Err)
				break
			}
			res := map[string]interface{}{
				"line":  result.Line,
				"error": result.Err.Error(),
			}
			mu.Lock()
			converted = append(converted, res)
			mu.Unlock()
			jsonBytes, _ := json.Marshal(res)
			_, _ = fmt.Fprintf(c.Writer, "%s\n", string(jsonBytes))
			flusher.Flush()
			continue
		}

		if result.Record != nil {
			stats.ReadLines++

			content, _ := result.Record.MarshalJSON()
			docxResult := services.ConvertRecordToDocx(content, fmt.Sprintf("record_%d.txt", result.Line), result.Line)

			if docxResult.Err != nil {
				stats.ErrorLines++
				slog.Errorf("convert record line %d to docx failed: %v", result.Line, docxResult.Err)
				res := map[string]interface{}{
					"line":  result.Line,
					"error": docxResult.Err.Error(),
				}
				mu.Lock()
				converted = append(converted, res)
				mu.Unlock()
				jsonBytes, _ := json.Marshal(res)
				_, _ = fmt.Fprintf(c.Writer, "%s\n", string(jsonBytes))
				flusher.Flush()
				continue
			}

			stats.ReturnLines++
			res := map[string]interface{}{
				"line":      result.Line,
				"file_name": docxResult.FileName,
				"file_path": docxResult.FilePath,
				"status":    "ok",
			}
			mu.Lock()
			converted = append(converted, res)
			mu.Unlock()
			jsonBytes, _ := json.Marshal(res)
			_, _ = fmt.Fprintf(c.Writer, "%s\n", string(jsonBytes))
			flusher.Flush()
		}
	}

	slog.Infof("JSONL convert finished: path=%s offset=%d limit=%d read=%d errors=%d converted=%d",
		validPath, offset, limit, stats.ReadLines, stats.ErrorLines, stats.ReturnLines)
}
