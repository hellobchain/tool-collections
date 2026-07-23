package services

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hellobchain/weekly-assistant/internal/constants"
	"github.com/hellobchain/weekly-assistant/internal/utils"
	"github.com/hellobchain/wswlog/wlogging"
	"github.com/ledongthuc/pdf"
	"github.com/richardlehane/mscfb"
)

var docLog = wlogging.MustGetLoggerWithoutName()

// ReadFile reads a file's content as plain text based on its extension.
// Supported formats: .txt, .docx, .doc, .pdf
func ReadFile(filename string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case constants.TxtExt:
		return string(data), nil
	case constants.DocxExt:
		return ReadDocx(data)
	case constants.DocExt:
		return ReadDoc(data)
	case constants.PdfExt:
		return ReadPDF(data)
	default:
		return string(data), nil
	}
}

// ReadDocx extracts text from a DOCX file (Office Open XML format).
// DOCX is a ZIP archive containing XML files; text is in word/document.xml.
func ReadDocx(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to open docx as zip: %w", err)
	}

	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open word/document.xml: %w", err)
			}
			defer rc.Close()

			xmlData, err := io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("failed to read word/document.xml: %w", err)
			}

			return extractDocxText(xmlData), nil
		}
	}

	return "", fmt.Errorf("word/document.xml not found in docx")
}

// docxBody represents the document structure for XML parsing
type docxBody struct {
	XMLName    xml.Name        `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main body"`
	Paragraphs []docxParagraph `xml:"p"`
}

type docxParagraph struct {
	Runs []docxRun `xml:"r"`
}

type docxRun struct {
	Text string `xml:"t"`
}

// extractDocxText parses the DOCX document XML and extracts text from <w:t> elements.
func extractDocxText(xmlData []byte) string {
	var body docxBody
	if err := xml.Unmarshal(xmlData, &body); err != nil {
		docLog.Warnf("Failed to parse docx XML, falling back to tag stripping: %v", err)
		return stripXMLTags(string(xmlData))
	}

	var parts []string
	for _, p := range body.Paragraphs {
		var lineParts []string
		for _, r := range p.Runs {
			lineParts = append(lineParts, r.Text)
		}
		parts = append(parts, strings.Join(lineParts, ""))
	}
	return strings.Join(parts, "\n")
}

// ReadDoc extracts text from a legacy .doc file (OLE2 / Compound Binary format).
func ReadDoc(data []byte) (string, error) {
	if len(data) < 8 {
		return "", fmt.Errorf("file too small to be a doc file")
	}

	// Some .doc files are actually .docx in disguise
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err == nil {
		return ReadDocx(data)
	}

	if !isOLE2(data) {
		return string(data), nil
	}

	doc, err := mscfb.New(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to parse OLE2: %w", err)
	}

	var wordStream []byte
	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		if entry.Name == "WordDocument" {
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, entry); err != nil {
				return "", fmt.Errorf("failed to read WordDocument stream: %w", err)
			}
			wordStream = buf.Bytes()
			break
		}
	}

	if len(wordStream) == 0 {
		return "", fmt.Errorf("WordDocument stream not found in doc file")
	}

	text := tryDecodeUTF16LE(wordStream)
	if text != "" {
		return text, nil
	}

	return extractPrintableStrings(wordStream), nil
}

// ReadPDF extracts text from a PDF file using the ledongthuc/pdf library.
func ReadPDF(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}

	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("failed to extract PDF text: %w", err)
	}

	text, err := io.ReadAll(textReader)
	if err != nil {
		return "", fmt.Errorf("failed to read PDF text: %w", err)
	}

	return string(text), nil
}

// stripXMLTags removes all XML tags from a string.
func stripXMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// isOLE2 checks if the data starts with the OLE2 magic bytes (D0CF11E0A1B11AE1).
func isOLE2(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0xD0 && data[1] == 0xCF &&
		data[2] == 0x11 && data[3] == 0xE0 &&
		data[4] == 0xA1 && data[5] == 0xB1 &&
		data[6] == 0x1A && data[7] == 0xE1
}

// tryDecodeUTF16LE attempts to decode data as UTF-16LE text segments.
func tryDecodeUTF16LE(data []byte) string {
	type seg struct{ start, end int }
	const minSeg = 6
	var segs []seg
	cur := -1

	for i := 0; i < len(data)-1; i += 2 {
		r := rune(data[i]) | rune(data[i+1])<<8
		if utils.IsTextRune(r) {
			if cur < 0 {
				cur = i
			}
		} else {
			if cur >= 0 && i-cur >= minSeg {
				if len(segs) > 0 && cur-segs[len(segs)-1].end <= 4 {
					segs[len(segs)-1].end = i
				} else {
					segs = append(segs, seg{cur, i})
				}
			}
			cur = -1
		}
	}
	if cur >= 0 && len(data)-cur >= minSeg {
		segs = append(segs, seg{cur, len(data)})
	}
	if len(segs) == 0 {
		return ""
	}

	var buf strings.Builder
	for _, s := range segs {
		raw := data[s.start:s.end]
		for i := 0; i < len(raw)-1; i += 2 {
			r := rune(raw[i]) | rune(raw[i+1])<<8
			if r == 0 || r == '\r' || r == '\n' {
				buf.WriteRune('\n')
			} else {
				buf.WriteRune(r)
			}
		}
		buf.WriteRune('\n')
	}
	return strings.TrimSpace(buf.String())
}

// extractPrintableStrings extracts sequences of printable ASCII characters.
func extractPrintableStrings(data []byte) string {
	var result strings.Builder
	var buf []byte

	for _, b := range data {
		if b >= 0x20 && b <= 0x7E {
			buf = append(buf, b)
		} else if b == '\n' || b == '\r' || b == '\t' {
			if len(buf) > 0 {
				buf = append(buf, b)
			}
		} else {
			if len(buf) >= 4 {
				if result.Len() > 0 {
					result.WriteRune('\n')
				}
				result.Write(buf)
			}
			buf = buf[:0]
		}
	}
	if len(buf) >= 4 {
		if result.Len() > 0 {
			result.WriteRune('\n')
		}
		result.Write(buf)
	}
	return result.String()
}
