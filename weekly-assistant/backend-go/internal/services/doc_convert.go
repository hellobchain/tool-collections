package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/hellobchain/weekly-assistant/internal/config"
)

type ConvertRequest struct {
	File     io.Reader
	FileName string
	ToFormat string
}

type ConvertResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Filename string   `json:"filename"`
		Formats  []string `json:"formats"`
		Document struct {
			Text string `json:"text,omitempty"`
			MD   string `json:"md,omitempty"`
			HTML string `json:"html,omitempty"`
			JSON string `json:"json,omitempty"`
		} `json:"document"`
	} `json:"data"`
}

func DocConvertFile(req *ConvertRequest) (*ConvertResponse, error) {
	cfg := config.AppConfig
	if !cfg.DocConvertEnable {
		return nil, fmt.Errorf("doc convert is not enabled")
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", req.FileName)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, req.File); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}

	if err := writer.WriteField("to_formats", req.ToFormat); err != nil {
		return nil, fmt.Errorf("write field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	httpReq, err := http.NewRequest("POST", cfg.DocConvertURL+cfg.DocConvertRouter, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+cfg.DocConvertAPIKey)
	httpReq.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: time.Duration(cfg.DocConvertTimeout) * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("convert failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result ConvertResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("convert failed: msg=%s", result.Msg)
	}

	return &result, nil
}
func DocConvertText(textBytes []byte, fileName string) (string, error) {
	var req = &ConvertRequest{
		File:     bytes.NewReader(textBytes),
		FileName: fileName,
		ToFormat: "text",
	}
	res, err := DocConvertFile(req)
	if err != nil {
		return "", err
	}
	return res.Data.Document.Text, nil
}
