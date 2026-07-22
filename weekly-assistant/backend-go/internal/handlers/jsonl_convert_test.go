package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/services"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func TestConvertJSONLToDocx_MissingFileParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/data/convert-to-docx", ConvertJSONLToDocx)

	body := models.JsonlReaderRequest{FilePath: ""}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/data/convert-to-docx", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp utils.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Code != utils.CodeInvalidParams {
		t.Fatalf("expected error code %d, got %d", utils.CodeInvalidParams, resp.Code)
	}
}

func TestConvertJSONLToDocx_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/data/convert-to-docx", ConvertJSONLToDocx)

	req := httptest.NewRequest("POST", "/api/v1/data/convert-to-docx", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp utils.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Code != utils.CodeInvalidParams {
		t.Fatalf("expected error code %d, got %d", utils.CodeInvalidParams, resp.Code)
	}
}

func TestConvertJSONLToDocx_FileNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/api/v1/data/convert-to-docx", ConvertJSONLToDocx)

	body := models.JsonlReaderRequest{FilePath: "C:/nonexistent.jsonl", Offset: 0, Limit: 10}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/data/convert-to-docx", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp utils.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Code != utils.CodeNotFound {
		t.Fatalf("expected error code %d, got %d", utils.CodeNotFound, resp.Code)
	}
}

func TestConvertJSONLToDocx_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir, _ := os.MkdirTemp("", "docx_test_*")
	defer os.RemoveAll(tmpDir)
	config.AppConfig = &config.Config{JSONLAllowedDir: "", LocalSavePath: tmpDir}

	r := gin.New()
	r.POST("/api/v1/data/convert-to-docx", ConvertJSONLToDocx)

	lines := []string{
		`{"name":"alice","role":"engineer"}`,
		`{"name":"bob","role":"designer"}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 0, Limit: 10}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/data/convert-to-docx", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	bodyStr := strings.TrimSpace(w.Body.String())
	gotLines := strings.Split(bodyStr, "\n")
	if len(gotLines) != 2 {
		t.Fatalf("expected 2 result lines, got %d: %s", len(gotLines), bodyStr)
	}

	for i, line := range gotLines {
		var res map[string]interface{}
		if err := json.Unmarshal([]byte(line), &res); err != nil {
			t.Fatalf("line %d: failed to parse result JSON: %v", i, err)
		}
		if res["status"] != "ok" {
			t.Fatalf("line %d: expected status ok, got %v", i, res["status"])
		}
		if res["file_name"] == nil {
			t.Fatalf("line %d: expected file_name", i)
		}
		fn := res["file_name"].(string)
		if !strings.HasSuffix(fn, ".docx") {
			t.Fatalf("line %d: expected .docx extension, got %s", i, fn)
		}
		if res["file_path"] == nil {
			t.Fatalf("line %d: expected file_path", i)
		}
		fp := res["file_path"].(string)
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			t.Fatalf("line %d: docx file not found on disk: %s", i, fp)
		}
		data, _ := os.ReadFile(fp)
		if len(data) == 0 {
			t.Fatalf("line %d: docx file is empty", i)
		}
	}
}

func TestConvertJSONLToDocx_WithOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir, _ := os.MkdirTemp("", "docx_test_*")
	defer os.RemoveAll(tmpDir)
	config.AppConfig = &config.Config{JSONLAllowedDir: "", LocalSavePath: tmpDir}

	r := gin.New()
	r.POST("/api/v1/data/convert-to-docx", ConvertJSONLToDocx)

	lines := []string{
		`{"id":1}`,
		`{"id":2}`,
		`{"id":3}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 1, Limit: 10}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/data/convert-to-docx", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	bodyStr := strings.TrimSpace(w.Body.String())
	gotLines := strings.Split(bodyStr, "\n")
	if len(gotLines) != 2 {
		t.Fatalf("expected 2 result lines (offset 1), got %d", len(gotLines))
	}
}

func TestConvertJSONLToDocx_WithLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir, _ := os.MkdirTemp("", "docx_test_*")
	defer os.RemoveAll(tmpDir)
	config.AppConfig = &config.Config{JSONLAllowedDir: "", LocalSavePath: tmpDir}

	r := gin.New()
	r.POST("/api/v1/data/convert-to-docx", ConvertJSONLToDocx)

	lines := []string{
		`{"id":1}`,
		`{"id":2}`,
		`{"id":3}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 0, Limit: 1}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/data/convert-to-docx", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	bodyStr := strings.TrimSpace(w.Body.String())
	gotLines := strings.Split(bodyStr, "\n")
	if len(gotLines) != 1 {
		t.Fatalf("expected 1 result line (limit 1), got %d", len(gotLines))
	}
}

func TestConvertJSONLToDocx_ContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: "", LocalSavePath: os.TempDir()}
	r := gin.New()
	r.POST("/api/v1/data/convert-to-docx", ConvertJSONLToDocx)

	lines := []string{`{"msg":"hello"}`}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 0, Limit: 10}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/data/convert-to-docx", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/x-ndjson" {
		t.Fatalf("expected Content-Type application/x-ndjson, got %s", ct)
	}
}

func TestMinimalDocx_IsValidZip(t *testing.T) {
	docxBytes := services.CreateMinimalDocx("test content")
	if len(docxBytes) == 0 {
		t.Fatal("expected non-empty docx bytes")
	}

	contentType := docxBytes[:4]
	if contentType[0] != 0x50 || contentType[1] != 0x4B ||
		contentType[2] != 0x03 || contentType[3] != 0x04 {
		t.Fatal("expected ZIP magic bytes (PK\\x03\\x04)")
	}
}

func TestStreamJSONL_ResponseEnds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	lines := []string{`{"id":1}`, `{"id":2}`, `{"id":3}`}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 0, Limit: 10}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `[DONE]`) {
		t.Fatal("expected [DONE] marker at end of stream")
	}
	// Body should end with the DONE marker (no trailing content after)
	if !strings.HasSuffix(strings.TrimSpace(bodyStr), `data: [DONE]`) {
		t.Fatalf("expected body to end with [DONE], got: %s", bodyStr[len(bodyStr)-50:])
	}
}
