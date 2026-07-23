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
	"golang.org/x/text/encoding/simplifiedchinese"
)

var docLog = wlogging.MustGetLoggerWithoutName()

// ReadFile reads a file's content as plain text based on its extension.
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

// ReadDocx extracts text from a DOCX file by parsing word/document.xml.
func ReadDocx(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty data")
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to open docx as zip: %w", err)
	}

	var firstNames []string
	for _, f := range reader.File {
		firstNames = append(firstNames, f.Name)
		name := f.Name
		// Normalize paths: some tools prefix with ./
		if len(name) > 2 && name[:2] == "./" {
			name = name[2:]
		}
		if name == "word/document.xml" {
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

	docLog.Warnf("DOCX zip entries (%d): %v", len(reader.File), firstNames)
	return "", fmt.Errorf("word/document.xml not found in docx (entries: %d)", len(reader.File))
}

type docxSimpleDoc struct {
	Body docxSimpleBody `xml:"body"`
}

type docxSimpleBody struct {
	Paragraphs []docxSimplePara `xml:"p"`
}

type docxSimplePara struct {
	Runs []docxSimpleRun `xml:"r"`
}

type docxSimpleRun struct {
	Text string `xml:"t"`
}

// extractDocxText strips the w: namespace prefix before parsing DOCX XML.
func extractDocxText(xmlData []byte) string {
	cleaned := stripDocxNS(xmlData)

	var doc docxSimpleDoc
	if err := xml.Unmarshal(cleaned, &doc); err != nil {
		docLog.Warnf("Failed to parse docx XML, falling back to tag stripping: %v", err)
		return stripXMLTags(string(xmlData))
	}

	var parts []string
	for _, p := range doc.Body.Paragraphs {
		var lineParts []string
		for _, r := range p.Runs {
			lineParts = append(lineParts, r.Text)
		}
		parts = append(parts, strings.Join(lineParts, ""))
	}
	return strings.Join(parts, "\n")
}

// stripDocxNS removes the w: namespace prefix and declarations from DOCX XML,
// so Go's encoding/xml can parse it without namespace-qualified struct tags.
func stripDocxNS(raw []byte) []byte {
	s := string(raw)
	s = strings.ReplaceAll(s, "w:", "")
	s = strings.ReplaceAll(s, `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`, "")
	s = strings.ReplaceAll(s, `xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"`, "")
	s = strings.ReplaceAll(s, `xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"`, "")
	s = strings.ReplaceAll(s, `xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"`, "")
	s = strings.ReplaceAll(s, `xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`, "")
	return []byte(s)
}

// ReadDoc extracts text from a legacy .doc file (OLE2 format).
func ReadDoc(data []byte) (string, error) {
	if len(data) < 8 {
		return "", fmt.Errorf("file too small to be a doc file")
	}

	// OLE2 (true .doc) must be checked BEFORE zip.NewReader, because
	// Go's zip reader scans the entire file for EOCD signature and
	// can false-positive on OLE2 binary data.
	if isOLE2(data) {
		return parseOLE2WordDoc(data)
	}

	// Some .doc files are actually renamed .docx (ZIP-based)
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err == nil {
		return ReadDocx(data)
	}

	return string(data), nil
}

func parseOLE2WordDoc(data []byte) (string, error) {
	doc, err := mscfb.New(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to parse OLE2: %w", err)
	}

	var wordStream, tableStream []byte
	for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
		switch entry.Name {
		case "WordDocument":
			buf := new(bytes.Buffer)
			io.Copy(buf, entry)
			wordStream = buf.Bytes()
		case "1Table", "0Table":
			buf := new(bytes.Buffer)
			io.Copy(buf, entry)
			tableStream = buf.Bytes()
		}
	}

	if len(wordStream) == 0 {
		return "", fmt.Errorf("WordDocument stream not found")
	}

	// Try piece-table-based extraction (most accurate)
	if text, ok := extractDocViaPieceTable(wordStream, tableStream); ok {
		return text, nil
	}

	// Fallback: skip FIB header (first 1024 bytes) and scan for UTF-16LE text
	// in WordDocument stream only (NOT 1Table/0Table which are binary noise).
	const fibHeaderSize = 1024
	textStart := wordStream
	if len(textStart) > fibHeaderSize {
		textStart = textStart[fibHeaderSize:]
	}

	text := tryDecodeUTF16LE(textStart)
	if len(text) > 20 {
		return text, nil
	}

	// Try GBK decoding for Chinese docs
	if len(textStart) > 0 {
		if decoded, err := tryDecodeGBK(textStart); err == nil && len(decoded) > 20 {
			return decoded, nil
		}
	}

	// Last resort: extract printable ASCII strings
	return extractPrintableStrings(wordStream), nil
}

// extractDocViaPieceTable parses the .doc piece table for accurate text extraction.
// Returns (text, true) on success.
func extractDocViaPieceTable(wordStream, tableStream []byte) (string, bool) {
	if len(wordStream) < 512 || len(tableStream) < 4 {
		return "", false
	}

	pieces := parsePieceTable(wordStream, tableStream)
	if len(pieces) == 0 {
		return "", false
	}

	var buf strings.Builder
	totalChars := 0
	for _, p := range pieces {
		for i := p.filePos; i+1 < p.filePos+p.length && i+1 < len(wordStream); i += 2 {
			r := rune(wordStream[i]) | rune(wordStream[i+1])<<8
			if r == 0 || r == '\r' {
				buf.WriteRune('\n')
			} else {
				buf.WriteRune(r)
			}
			totalChars++
		}
	}

	if totalChars < 10 {
		return "", false
	}
	return strings.TrimSpace(buf.String()), true
}

type pieceDescriptor struct {
	filePos int
	length  int
}

// parsePieceTable extracts the piece table from the 1Table/0Table stream.
// The CLX (complex) contains the Pcdt (piece table) which maps character
// positions to file positions in the WordDocument stream.
func parsePieceTable(wordStream, tableStream []byte) []pieceDescriptor {
	// First, try to find fcClx/lcbClx from the FIB in WordDocument stream.
	// FIB base: fcClx at offset 0x004A (4 bytes), lcbClx at offset 0x004E (4 bytes).
	clxOff := int(readU32LE(wordStream, 0x004A))
	clxLen := int(readU32LE(wordStream, 0x004E))

	if clxOff <= 0 || clxLen <= 4 || clxOff+clxLen > len(tableStream) {
		// Fallback: scan tableStream for CLX marker (0x01 = clxtPcdt)
		clxOff = scanForCLX(tableStream)
		if clxOff < 0 {
			return nil
		}
		clxLen = len(tableStream) - clxOff
	}

	clx := tableStream[clxOff : clxOff+clxLen]
	return parsePcdt(clx, wordStream)
}

// readU32LE reads a 32-bit unsigned little-endian integer from data at offset.
func readU32LE(data []byte, off int) uint32 {
	if off+4 > len(data) {
		return 0
	}
	return uint32(data[off]) | uint32(data[off+1])<<8 | uint32(data[off+2])<<16 | uint32(data[off+3])<<24
}

// readU16LE reads a 16-bit unsigned little-endian integer from data at offset.
func readU16LE(data []byte, off int) uint16 {
	if off+2 > len(data) {
		return 0
	}
	return uint16(data[off]) | uint16(data[off+1])<<8
}

const (
	clxtPcdt = 0x01 // CLX type: Piece Table
)

// scanForCLX looks for the CLX (complex) marker in the table stream.
func scanForCLX(data []byte) int {
	for i := 0; i < len(data)-5; i++ {
		if data[i] == clxtPcdt {
			// Verify: next 4 bytes should be a reasonable size
			lcb := int(readU32LE(data, i+1))
			if lcb > 0 && lcb < len(data)-i-5 {
				return i
			}
		}
	}
	return -1
}

// parsePcdt parses the Pcdt (Piece Table) from a CLX blob.
func parsePcdt(clx []byte, wordStream []byte) []pieceDescriptor {
	if len(clx) < 5 || clx[0] != clxtPcdt {
		return nil
	}

	lcb := int(readU32LE(clx, 1))
	if lcb < 8 || lcb+5 > len(clx) {
		return nil
	}

	data := clx[5 : 5+lcb]

	// Pcdt structure:
	//   ccp (4 bytes): count of character positions = number of pieces + 1
	//   rgcp (ccp * 4 bytes): array of character positions (CP)
	//   rgPcd ((ccp-1) * 8 bytes): array of Pcd (8 bytes each)

	ccp := int(readU32LE(data, 0))
	if ccp < 2 || ccp > 100000 {
		return nil
	}

	// rgcp starts at offset 4, rgPcd starts at offset 4 + ccp*4
	rgcpOff := 4
	pcdOff := rgcpOff + ccp*4

	if pcdOff+8 > len(data) {
		return nil
	}

	numPieces := ccp - 1
	pieces := make([]pieceDescriptor, 0, numPieces)

	for i := 0; i < numPieces; i++ {
		pcdStart := pcdOff + i*8
		if pcdStart+8 > len(data) {
			break
		}

		// Pcd structure (8 bytes):
		//   fc (4 bytes): bit 31 = fCompressed, bit 30 = reserved
		//                 bits 0-29 = file position in WordDocument
		//   prm (2 bytes)
		//   reserved (2 bytes)
		fcRaw := readU32LE(data, pcdStart)

		fCompressed := (fcRaw >> 31) & 1
		filePos := int(fcRaw & 0x3FFFFFFF) // 30 bits

		// Get character count from adjacent CP positions
		cpStart := int(readU32LE(data, rgcpOff+i*4))
		cpEnd := int(readU32LE(data, rgcpOff+(i+1)*4))
		charCount := cpEnd - cpStart

		if filePos < 0 || charCount <= 0 || filePos >= len(wordStream) {
			continue
		}

		if fCompressed == 0 {
			// Uncompressed (UTF-16LE): each CP = 2 bytes
			byteLen := charCount * 2
			if filePos+byteLen > len(wordStream) {
				byteLen = len(wordStream) - filePos
			}
			if byteLen > 0 {
				pieces = append(pieces, pieceDescriptor{
					filePos: filePos,
					length:  byteLen,
				})
			}
		} else {
			// Compressed (single byte per char, usually ANSI codepage)
			byteLen := charCount
			if filePos+byteLen > len(wordStream) {
				byteLen = len(wordStream) - filePos
			}
			if byteLen > 0 {
				pieces = append(pieces, pieceDescriptor{
					filePos: filePos,
					length:  byteLen,
				})
			}
		}
	}

	return pieces
}

var gbkEncoder = simplifiedchinese.GB18030.NewDecoder()

// tryDecodeGBK attempts to decode data as GBK/GB18030.
// It returns the decoded text if successful and looks like real text.
func tryDecodeGBK(data []byte) (string, error) {
	decoded, err := gbkEncoder.Bytes(data)
	if err != nil {
		return "", err
	}
	s := string(decoded)
	// Check if result looks like real text
	if len(s) < 20 {
		return "", fmt.Errorf("too short")
	}
	printable := 0
	for _, r := range s {
		if r >= 0x20 && r <= 0x7E || r >= 0x4E00 && r <= 0x9FFF || r == '\n' || r == '\t' {
			printable++
		}
	}
	if printable < len(s)/2 {
		return "", fmt.Errorf("too many non-printable characters")
	}
	return s, nil
}

// ReadPDF extracts text from a PDF file.
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

func isOLE2(data []byte) bool {
	return len(data) >= 8 &&
		data[0] == 0xD0 && data[1] == 0xCF &&
		data[2] == 0x11 && data[3] == 0xE0 &&
		data[4] == 0xA1 && data[5] == 0xB1 &&
		data[6] == 0x1A && data[7] == 0xE1
}

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
