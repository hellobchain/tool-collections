package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GenericRecord struct {
	Raw json.RawMessage
}

func (r *GenericRecord) MarshalJSON() ([]byte, error) {
	return r.Raw, nil
}

func (r *GenericRecord) UnmarshalJSON(data []byte) error {
	r.Raw = make(json.RawMessage, len(data))
	copy(r.Raw, data)
	return nil
}

func (r *GenericRecord) ToMap() (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(r.Raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

type StreamResult struct {
	Record *GenericRecord
	Line   int
	Err    error
}

type StreamStats struct {
	ReadLines   int `json:"read_lines"`
	ErrorLines  int `json:"error_lines"`
	ReturnLines int `json:"return_lines"`
}

func ValidateFilePath(baseDir, requestedPath string) (string, error) {
	clean := filepath.Clean(requestedPath)
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("path must be absolute: %s", requestedPath)
	}
	if baseDir != "" {
		rel, err := filepath.Rel(baseDir, clean)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("access denied: path %s is outside allowed directory %s", requestedPath, baseDir)
		}
	}
	return clean, nil
}

func ReadJSONLStream(ctx context.Context, filePath string, offset, limit int, results chan<- StreamResult) {
	defer close(results)

	f, err := os.Open(filePath)
	if err != nil {
		results <- StreamResult{Err: fmt.Errorf("failed to open file %s: %w", filePath, err)}
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	lineNum := 0
	returned := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			slog.Warnf("JSONL stream cancelled at line %d: %v", lineNum, ctx.Err())
			return
		default:
		}

		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			slog.Warnf("empty line at line %d", lineNum)
			continue
		}

		if lineNum <= offset {
			slog.Warnf("skipping line %d", lineNum)
			continue
		}

		if limit > 0 && returned >= limit {
			slog.Warnf("reached limit of %d lines", limit)
			break
		}
		raw := json.RawMessage(line)
		if !json.Valid([]byte(line)) {
			slog.Warnf("invalid JSON at line %d", lineNum)
			slog.Errorf("failed to parse JSONL at line %d: invalid JSON", lineNum)
			results <- StreamResult{Line: lineNum, Err: fmt.Errorf("parse error at line %d: invalid JSON", lineNum)}
			returned++
			continue
		}

		rec := &GenericRecord{Raw: raw}
		results <- StreamResult{Record: rec, Line: lineNum}
		returned++
	}

	if err := scanner.Err(); err != nil {
		results <- StreamResult{Err: fmt.Errorf("I/O error reading %s: %w", filePath, err)}
	}
}

func ReadJSONLStreamWithSchema(ctx context.Context, filePath string, offset, limit int, schema map[string]interface{}, results chan<- StreamResult) {
	defer close(results)

	f, err := os.Open(filePath)
	if err != nil {
		results <- StreamResult{Err: fmt.Errorf("failed to open file %s: %w", filePath, err)}
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	lineNum := 0
	returned := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			slog.Warnf("JSONL stream cancelled at line %d: %v", lineNum, ctx.Err())
			return
		default:
		}

		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			slog.Warnf("empty line at line %d", lineNum)
			continue
		}

		if lineNum <= offset {
			slog.Warnf("skipping line %d", lineNum)
			continue
		}

		if limit > 0 && returned >= limit {
			slog.Warnf("reached limit of %d lines", limit)
			break
		}

		if !json.Valid([]byte(line)) {
			slog.Errorf("failed to parse JSONL at line %d: invalid JSON", lineNum)
			results <- StreamResult{Line: lineNum, Err: fmt.Errorf("parse error at line %d: invalid JSON", lineNum)}
			returned++
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			slog.Errorf("failed to parse JSONL at line %d: %v", lineNum, err)
			results <- StreamResult{Line: lineNum, Err: fmt.Errorf("parse error at line %d: %w", lineNum, err)}
			returned++
			continue
		}

		for key := range schema {
			if _, ok := data[key]; !ok {
				slog.Warnf("schema validation warning at line %d: missing field '%s'", lineNum, key)
			}
		}

		raw := json.RawMessage(line)
		rec := &GenericRecord{Raw: raw}
		results <- StreamResult{Record: rec, Line: lineNum}
		returned++
	}

	if err := scanner.Err(); err != nil {
		results <- StreamResult{Err: fmt.Errorf("I/O error reading %s: %w", filePath, err)}
	}
}

func CountFileLines(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}
	return lineCount, scanner.Err()
}

func WriteTempJSONL(records []interface{}) (string, error) {
	f, err := os.CreateTemp("", "jsonl_test_*.jsonl")
	if err != nil {
		return "", err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			os.Remove(f.Name())
			return "", err
		}
	}
	return f.Name(), nil
}

func FileExists(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return info, nil
}
