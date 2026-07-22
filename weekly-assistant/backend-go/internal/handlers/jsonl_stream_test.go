package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/weekly-assistant/internal/utils"
)

func TestStreamJSONL_MissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/data/stream", StreamJSONL)

	req := httptest.NewRequest("GET", "/api/v1/data/stream", nil)
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
	r.GET("/api/v1/data/stream", StreamJSONL)

	req := httptest.NewRequest("GET", "/api/v1/data/stream?file=C:/nonexistent_test_file_12345.jsonl", nil)
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
	r.GET("/api/v1/data/stream", StreamJSONL)

	req := httptest.NewRequest("GET", "/api/v1/data/stream?file=C:/test.jsonl&offset=-1", nil)
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

func TestStreamJSONL_InvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/data/stream", StreamJSONL)

	req := httptest.NewRequest("GET", "/api/v1/data/stream?file=C:/test.jsonl&limit=abc", nil)
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

func TestStreamJSONL_LimitExceedsMax(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/data/stream", StreamJSONL)

	req := httptest.NewRequest("GET", "/api/v1/data/stream?file=C:/test.jsonl&limit=99999", nil)
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
	r.GET("/api/v1/data/stream", StreamJSONL)

	lines := []string{
		`{"name":"alice","age":30,"tags":["dev","go"]}`,
		`{"name":"bob","age":25,"active":true}`,
		`{"product":"widget","price":9.99,"stock":100}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/data/stream?file=%s", tmpFile), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"name":"alice"`) {
		t.Fatal("expected response to contain first record")
	}
	if !strings.Contains(body, `"product":"widget"`) {
		t.Fatal("expected response to contain third record with different schema")
	}
	if !strings.Contains(body, `"done"`) {
		t.Fatal("expected response to contain done message")
	}
}

func TestStreamJSONL_WithSchema(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.GET("/api/v1/data/stream", StreamJSONL)

	lines := []string{
		`{"id":1,"value":"a"}`,
		`{"id":2,"value":"b","extra":"x"}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	schema := `{"id":null,"value":null}`
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/data/stream?file=%s&schema=%s", tmpFile, schema), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"id":1`) {
		t.Fatal("expected response to contain id field")
	}
	if !strings.Contains(body, `"done"`) {
		t.Fatal("expected done message")
	}
}

func TestStreamJSONL_WithOffset(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.GET("/api/v1/data/stream", StreamJSONL)

	lines := []string{
		`{"id":1}`,
		`{"id":2}`,
		`{"id":3}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/data/stream?file=%s&offset=1", tmpFile), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if strings.Contains(body, `"id":1`) {
		t.Fatal("expected offset 1 to skip first record")
	}
	if !strings.Contains(body, `"id":3`) {
		t.Fatal("expected offset 1 to include third record")
	}
}

func TestStreamJSONL_WithLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.GET("/api/v1/data/stream", StreamJSONL)

	lines := []string{
		`{"id":1}`,
		`{"id":2}`,
		`{"id":3}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/data/stream?file=%s&limit=2", tmpFile), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `"id":1`) {
		t.Fatal("expected limit to include first record")
	}
	if strings.Contains(body, `"id":3`) {
		t.Fatal("expected limit 2 to exclude third record")
	}
}

func TestStreamJSONL_SSEHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.AppConfig = &config.Config{JSONLAllowedDir: ""}
	r := gin.New()
	r.GET("/api/v1/data/stream", StreamJSONL)

	lines := []string{`{"msg":"hello"}`}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/data/stream?file=%s", tmpFile), nil)
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
	r.GET("/api/v1/data/stream", StreamJSONL)

	req := httptest.NewRequest("GET", "/api/v1/data/stream?file=C:/notallowed/test.jsonl", nil)
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
	r.GET("/api/v1/data/stream", StreamJSONL)

	tmpFile := writeLinesToTempFile(t, []string{`{"id":1}`})
	defer os.Remove(tmpFile)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/data/stream?file=%s&schema=not-json", tmpFile), nil)
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
	r.GET("/api/v1/data/stream", StreamJSONL)

	lines := []string{
		`{"string":"hello","number":42,"bool":true,"null":null,"arr":[1,2,3],"obj":{"nested":"value"}}`,
		`[1,2,3]`,
		`"just a string"`,
		`12345`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/data/stream?file=%s", tmpFile), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"nested":"value"`) {
		t.Fatal("expected nested object to be preserved")
	}
	if !strings.Contains(body, `[1,2,3]`) {
		t.Fatal("expected array JSON to be supported")
	}
	if !strings.Contains(body, `"just a string"`) {
		t.Fatal("expected string JSON to be supported")
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
