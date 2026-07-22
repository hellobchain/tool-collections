package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFilePath_AbsoluteNoBase(t *testing.T) {
	path, err := ValidateFilePath("", "C:/data/test.jsonl")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != "C:\\data\\test.jsonl" {
		t.Fatalf("expected C:\\data\\test.jsonl, got %s", path)
	}
}

func TestValidateFilePath_RelativeRejected(t *testing.T) {
	_, err := ValidateFilePath("", "relative/path.jsonl")
	if err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestValidateFilePath_WithinBaseDir(t *testing.T) {
	path, err := ValidateFilePath("C:/data", "C:/data/sub/test.jsonl")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatal("expected absolute path")
	}
}

func TestValidateFilePath_OutsideBaseDir(t *testing.T) {
	_, err := ValidateFilePath("C:/data", "C:/other/outside.jsonl")
	if err == nil {
		t.Fatal("expected error for path outside base dir")
	}
}

func TestValidateFilePath_TraversalAttack(t *testing.T) {
	_, err := ValidateFilePath("C:/data", "C:/data/../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestReadJSONLStream_Basic(t *testing.T) {
	lines := []string{
		`{"id":"1","time":1000,"data":{"key":"value1"}}`,
		`{"id":"2","time":2000,"data":{"key":"value2"}}`,
		`{"id":"3","time":3000,"data":{"key":"value3"}}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStream(ctx, tmpFile, 0, 0, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Fatalf("unexpected error at %d: %v", i, r.Err)
		}
		if r.Record == nil {
			t.Fatalf("expected record at %d", i)
		}
	}
}

func TestReadJSONLStream_Offset(t *testing.T) {
	lines := []string{
		`{"id":"1"}`,
		`{"id":"2"}`,
		`{"id":"3"}`,
		`{"id":"4"}`,
		`{"id":"5"}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStream(ctx, tmpFile, 2, 0, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results (offset 2), got %d", len(results))
	}
}

func TestReadJSONLStream_Limit(t *testing.T) {
	lines := []string{
		`{"id":"1"}`,
		`{"id":"2"}`,
		`{"id":"3"}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStream(ctx, tmpFile, 0, 2, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results (limit 2), got %d", len(results))
	}
}

func TestReadJSONLStream_OffsetAndLimit(t *testing.T) {
	lines := []string{
		`{"id":"1"}`,
		`{"id":"2"}`,
		`{"id":"3"}`,
		`{"id":"4"}`,
		`{"id":"5"}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStream(ctx, tmpFile, 1, 2, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results (offset 1, limit 2), got %d", len(results))
	}
}

func TestReadJSONLStream_FileNotFound(t *testing.T) {
	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStream(ctx, "C:/nonexistent/file.jsonl", 0, 10, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 error result, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestReadJSONLStream_EmptyFile(t *testing.T) {
	tmpFile := writeLinesToTempFile(t, []string{})
	defer os.Remove(tmpFile)

	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStream(ctx, tmpFile, 0, 10, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty file, got %d", len(results))
	}
}

func TestReadJSONLStream_EmptyLines(t *testing.T) {
	lines := []string{"", "", `{"id":"1"}`, "", `{"id":"2"}`}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStream(ctx, tmpFile, 0, 10, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results (skipping empty lines), got %d", len(results))
	}
}

func TestReadJSONLStream_ContextCancellation(t *testing.T) {
	lines := make([]string, 100)
	for i := 0; i < 100; i++ {
		lines[i] = `{"id":1}`
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	ch := make(chan StreamResult, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go ReadJSONLStream(ctx, tmpFile, 0, 0, ch)

	read := 0
	for r := range ch {
		if r.Err != nil {
			t.Fatalf("unexpected error: %v", r.Err)
		}
		read++
		if read >= 5 {
			cancel()
			break
		}
	}

	if read < 5 {
		t.Fatalf("expected at least 5 records before cancellation, got %d", read)
	}
}

func TestReadJSONLStream_InvalidJSON(t *testing.T) {
	lines := []string{
		`{"valid":true}`,
		`not valid json`,
		`{"valid":true}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStream(ctx, tmpFile, 0, 0, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results (2 good + 1 error), got %d", len(results))
	}
	if results[0].Record == nil {
		t.Fatal("expected first record to be valid")
	}
	if results[1].Err == nil {
		t.Fatal("expected second line to produce error")
	}
	if results[2].Record == nil {
		t.Fatal("expected third record to be valid")
	}
}

func TestGenericRecord_ToMap(t *testing.T) {
	rec := &GenericRecord{Raw: json.RawMessage(`{"name":"test","count":5}`)}
	m, err := rec.ToMap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["name"] != "test" {
		t.Fatalf("expected name='test', got %v", m["name"])
	}
	if m["count"].(float64) != 5 {
		t.Fatalf("expected count=5, got %v", m["count"])
	}
}

func TestGenericRecord_MarshalJSON(t *testing.T) {
	raw := json.RawMessage(`{"a":1,"b":"x"}`)
	rec := &GenericRecord{Raw: raw}

	data, err := rec.MarshalJSON()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"a":1,"b":"x"}` {
		t.Fatalf("expected original JSON, got %s", string(data))
	}
}

func TestReadJSONLStreamWithSchema_Basic(t *testing.T) {
	lines := []string{
		`{"id":1,"name":"alice"}`,
		`{"id":2,"name":"bob","extra":"x"}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	schema := map[string]interface{}{"id": nil, "name": nil}
	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStreamWithSchema(ctx, tmpFile, 0, 0, schema, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestReadJSONLStreamWithSchema_MissingFields(t *testing.T) {
	lines := []string{
		`{"id":1}`,
		`{"id":2,"name":"bob"}`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	schema := map[string]interface{}{"id": nil, "name": nil}
	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStreamWithSchema(ctx, tmpFile, 0, 0, schema, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestReadJSONLStreamWithSchema_InvalidJSON(t *testing.T) {
	lines := []string{
		`{"id":1}`,
		`bad json`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	schema := map[string]interface{}{"id": nil}
	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStreamWithSchema(ctx, tmpFile, 0, 0, schema, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results (1 good + 1 error), got %d", len(results))
	}
	if results[0].Record == nil {
		t.Fatal("expected first record to be valid")
	}
	if results[1].Err == nil {
		t.Fatal("expected second line to produce error")
	}
}

func TestCountFileLines(t *testing.T) {
	lines := make([]string, 50)
	for i := 0; i < 50; i++ {
		lines[i] = `{"id":1}`
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	count, err := CountFileLines(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 50 {
		t.Fatalf("expected 50 lines, got %d", count)
	}
}

func TestReadJSONLStream_VariousJSONTypes(t *testing.T) {
	lines := []string{
		`{"string":"hello","number":42,"bool":true,"null":null,"arr":[1,2,3],"obj":{"nested":"v"}}`,
		`[1,2,3]`,
		`"plain string"`,
		`42`,
		`true`,
		`null`,
	}
	tmpFile := writeLinesToTempFile(t, lines)
	defer os.Remove(tmpFile)

	ch := make(chan StreamResult, 10)
	ctx := context.Background()
	go ReadJSONLStream(ctx, tmpFile, 0, 0, ch)

	var results []StreamResult
	for r := range ch {
		results = append(results, r)
	}

	if len(results) != 6 {
		t.Fatalf("expected 6 results for various JSON types, got %d", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Fatalf("unexpected error at index %d (line %d): %v", i, r.Line, r.Err)
		}
		if r.Record == nil || len(r.Record.Raw) == 0 {
			t.Fatalf("expected non-empty record at index %d", i)
		}
	}
}

func TestGenericRecord_UnmarshalPreservesRaw(t *testing.T) {
	input := `{"a":1,"b":"hello"}`
	var rec GenericRecord
	if err := json.Unmarshal([]byte(input), &rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(rec.Raw) != input {
		t.Fatalf("expected raw '%s', got '%s'", input, string(rec.Raw))
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
