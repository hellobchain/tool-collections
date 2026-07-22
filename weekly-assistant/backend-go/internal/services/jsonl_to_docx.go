package services

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hellobchain/weekly-assistant/internal/config"
	"github.com/hellobchain/wswlog/wlogging"
)

var docxSlog = wlogging.MustGetLoggerWithoutName()

type DocxConvertResult struct {
	FilePath string
	FileName string
	Line     int
	Err      error
}

func ConvertRecordToDocx(content []byte, fileName string, lineNum int) *DocxConvertResult {
	cfg := config.AppConfig

	docxBody := &bytes.Buffer{}
	writer := multipart.NewWriter(docxBody)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return &DocxConvertResult{Line: lineNum, Err: fmt.Errorf("create form file: %w", err)}
	}
	if _, err := part.Write(content); err != nil {
		return &DocxConvertResult{Line: lineNum, Err: fmt.Errorf("write file: %w", err)}
	}

	if err := writer.WriteField("to_formats", "docx"); err != nil {
		return &DocxConvertResult{Line: lineNum, Err: fmt.Errorf("write field: %w", err)}
	}

	if err := writer.Close(); err != nil {
		return &DocxConvertResult{Line: lineNum, Err: fmt.Errorf("close writer: %w", err)}
	}

	httpReq, err := http.NewRequest("POST", cfg.DocConvertURL+cfg.DocConvertRouter, docxBody)
	if err != nil {
		return &DocxConvertResult{Line: lineNum, Err: fmt.Errorf("create request: %w", err)}
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Authorization", "Bearer "+cfg.DocConvertAPIKey)
	httpReq.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: time.Duration(cfg.DocConvertTimeout) * time.Second}
	resp, err := client.Do(httpReq)

	var docxData []byte
	if err != nil {
		docxSlog.Warnf("doc conversion service unavailable, using fallback docx for line %d: %v", lineNum, err)
		docxData = CreateMinimalDocx(string(content))
	} else {
		defer resp.Body.Close()
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			docxSlog.Warnf("failed to read doc conversion response, using fallback for line %d: %v", lineNum, readErr)
			docxData = CreateMinimalDocx(string(content))
		} else if resp.StatusCode != http.StatusOK {
			docxSlog.Warnf("doc conversion returned status %d, using fallback for line %d", resp.StatusCode, lineNum)
			docxData = CreateMinimalDocx(string(content))
		} else {
			docxData, _ = extractDocxBytes(respBody, content)
		}
	}

	outName := fmt.Sprintf("%s.docx", uuid.New().String())
	outPath := cfg.LocalSavePath + "/" + outName

	if err := writeFile(outPath, docxData); err != nil {
		return &DocxConvertResult{Line: lineNum, Err: fmt.Errorf("write docx: %w", err)}
	}

	docxSlog.Infof("JSONL record converted to docx: line=%d file=%s", lineNum, outPath)
	return &DocxConvertResult{
		FilePath: outPath,
		FileName: outName,
		Line:     lineNum,
	}
}

func extractDocxBytes(respBody, originalContent []byte) ([]byte, error) {
	var convResp ConvertResponse
	if err := json.Unmarshal(respBody, &convResp); err == nil && convResp.Code == 0 {
		text := convResp.Data.Document.Text
		if text == "" {
			text = convResp.Data.Document.MD
		}
		if text == "" {
			text = convResp.Data.Document.HTML
		}
		if text == "" {
			text = convResp.Data.Document.JSON
		}
		if text != "" {
			return CreateMinimalDocx(text), nil
		}
	}

	if isLikelyDocx(respBody) {
		return respBody, nil
	}

	return CreateMinimalDocx(string(originalContent)), nil
}

func isLikelyDocx(data []byte) bool {
	return len(data) > 4 &&
		data[0] == 0x50 && data[1] == 0x4B &&
		data[2] == 0x03 && data[3] == 0x04
}

func CreateMinimalDocx(text string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`
	writeZipEntry(w, "[Content_Types].xml", contentTypes)

	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
	writeZipEntry(w, "_rels/.rels", rels)

	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`
	writeZipEntry(w, "word/_rels/document.xml.rels", docRels)

	escapedText := escapeXML(text)
	document := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>%s</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`, escapedText)
	writeZipEntry(w, "word/document.xml", document)

	styles := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal">
    <w:name w:val="Normal"/>
  </w:style>
</w:styles>`
	writeZipEntry(w, "word/styles.xml", styles)

	w.Close()
	return buf.Bytes()
}

func writeZipEntry(w *zip.Writer, name, content string) {
	f, _ := w.Create(name)
	f.Write([]byte(content))
}

func escapeXML(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch r {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
