package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/models"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func TestStreamJSONL_MissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	body := models.JsonlReaderRequest{FilePath: ""}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
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

func TestStreamJSONL_FileNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	body := models.JsonlReaderRequest{FilePath: "C:/nonexistent.jsonl"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
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

func TestStreamJSONL_InvalidOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	body := models.JsonlReaderRequest{FilePath: "C:/test.jsonl", Offset: -1}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
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

func TestStreamJSONL_InvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	body := models.JsonlReaderRequest{FilePath: "C:/test.jsonl", Limit: -1}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
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

func TestStreamJSONL_LimitExceedsMax(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	body := models.JsonlReaderRequest{FilePath: "C:/test.jsonl", Limit: 99999}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
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

func TestStreamJSONL_Success_GenericJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	lines := []string{
		`{"name":"alice","age":30,"tags":["dev","go"]}`,
		`{"name":"bob","age":25,"active":true}`,
		`{"product":"widget","price":9.99,"stock":100}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 0, Limit: 10}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `"name":"alice"`) {
		t.Fatal("expected response to contain first record")
	}
	if !strings.Contains(bodyStr, `"product":"widget"`) {
		t.Fatal("expected response to contain third record")
	}
	if !strings.Contains(bodyStr, `[DONE]`) {
		t.Fatal("expected response to contain [DONE] marker")
	}
}

func TestStreamJSONL_WithSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	lines := []string{
		`{"id":1,"value":"a"}`,
		`{"id":2,"value":"b","extra":"x"}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 0, Limit: 10, Schema: `{"id":null,"value":null}`}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `"id":1`) {
		t.Fatal("expected response to contain id=1")
	}
	if !strings.Contains(bodyStr, `[DONE]`) {
		t.Fatal("expected [DONE] marker")
	}
}

func TestStreamJSONL_WithOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	lines := []string{`{"id":1}`, `{"id":2}`, `{"id":3}`}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 1, Limit: 10}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, `"id":1`) {
		t.Fatal("expected offset 1 to skip first record")
	}
	if !strings.Contains(bodyStr, `"id":3`) {
		t.Fatal("expected offset 1 to include third record")
	}
	if !strings.Contains(bodyStr, `[DONE]`) {
		t.Fatal("expected [DONE] marker")
	}
}

func TestStreamJSONL_WithLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	lines := []string{`{"id":1}`, `{"id":2}`, `{"id":3}`}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 0, Limit: 2}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, `"id":3`) {
		t.Fatal("expected limit 2 to exclude third record")
	}
	if !strings.Contains(bodyStr, `[DONE]`) {
		t.Fatal("expected [DONE] marker")
	}
}

func TestStreamJSONL_ContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	lines := []string{`{"msg":"hello"}`}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %s", ct)
	}
}

func TestStreamJSONL_AllowedDirProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: "C:/allowed"}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	body := models.JsonlReaderRequest{FilePath: "C:/notallowed/test.jsonl"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
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

func TestStreamJSONL_InvalidSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	lines := []string{`{"id":1}`}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Schema: "not-json"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
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

func TestStreamJSONL_DifferentJSONStructures(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.POST("/jsonl-read/v1/data/stream", StreamJSONL)

	lines := []string{
		`{"string":"hello","number":42,"bool":true,"null":null,"arr":[1,2,3],"obj":{"nested":"value"}}`,
		`[1,2,3]`,
		`"just a string"`,
		`12345`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	body := models.JsonlReaderRequest{FilePath: tmpFile, Offset: 0, Limit: 10}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jsonl-read/v1/data/stream", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, `"nested":"value"`) {
		t.Fatal("expected nested object to be preserved")
	}
	if !strings.Contains(bodyStr, `[1,2,3]`) {
		t.Fatal("expected array JSON to be supported")
	}
	if !strings.Contains(bodyStr, `"just a string"`) {
		t.Fatal("expected string JSON to be supported")
	}
	if !strings.Contains(bodyStr, `[DONE]`) {
		t.Fatal("expected [DONE] marker")
	}
}

func writeLinesToTempFile(t *testing.T, lines []string) string {
	t.Helper()
	f, err := os.CreateTemp("", "jsonl_test_*.jsonl")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	for _, line := range lines {
		fmt.Fprintln(f, line)
	}
	f.Close()
	return f.Name()
}
