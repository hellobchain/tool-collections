package services

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hellobchain/weekly-assistant/internal/constants"
	"github.com/hellobchain/wswlog/wlogging"
	"github.com/ledongthuc/pdf"
	"github.com/richardlehane/mscfb"
)

var mdLog = wlogging.MustGetLoggerWithoutName()

// Markitdown converts various file formats to Markdown.
type Markitdown struct{}

// Result holds the Markdown conversion result.
type Result struct {
	Title       string
	Content     string
	TextContent string
}

// Convert converts a file's content to Markdown based on its extension.
func (m *Markitdown) Convert(filename string, data []byte) (*Result, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case constants.TxtExt:
		return m.convertText(data)
	case constants.DocxExt:
		return m.convertDocx(data)
	case constants.DocExt:
		return m.convertDoc(data)
	case constants.PdfExt:
		return m.convertPDF(data)
	case constants.HtmlExt, constants.HtmExt:
		return m.convertText(data)
	default:
		return &Result{Content: string(data), TextContent: string(data)}, nil
	}
}

func (m *Markitdown) convertText(data []byte) (*Result, error) {
	text := string(data)
	return &Result{Content: text, TextContent: text}, nil
}

// --- DOCX Conversion ---

// All DOCX structs are parsed after namespace stripping, so tags don't need namespace URLs.

type docxStyles struct {
	XMLName xml.Name        `xml:"styles"`
	Styles  []docxStyleInfo `xml:"style"`
}

type docxStyleInfo struct {
	StyleID    string `xml:"styleId,attr"`
	Type       string `xml:"type,attr"`
	Name       string `xml:"name>val"`
	OutlineLvl int    `xml:"pPr>outlineLvl>val"`
}

type docxDocument struct {
	XMLName xml.Name     `xml:"document"`
	Body    docxBodyFull `xml:"body"`
}

type docxBodyFull struct {
	Paragraphs []docxParagraphFull `xml:"p"`
	Tables     []docxTable         `xml:"tbl"`
}

type docxParagraphFull struct {
	PPr        *docxPPr        `xml:"pPr"`
	Runs       []docxRunFull   `xml:"r"`
	Hyperlinks []docxHyperlink `xml:"hyperlink"`
}

type docxPPr struct {
	StyleID       string     `xml:"pStyle>val"`
	NumPr         *docxNumPr `xml:"numPr"`
	Justification string     `xml:"jc>val"`
}

type docxNumPr struct {
	Ilvl  string `xml:"ilvl>val"`
	NumID string `xml:"numId>val"`
}

type docxRunFull struct {
	RPr  *docxRPr `xml:"rPr"`
	Text string   `xml:"t"`
}

type docxRPr struct {
	Bold      *struct{} `xml:"b"`
	BoldVal   string    `xml:"b>val"`
	Italic    *struct{} `xml:"i"`
	ItalicVal string    `xml:"i>val"`
	Underline *struct{} `xml:"u"`
	Color     string    `xml:"color>val"`
	RFonts    string    `xml:"rFonts>ascii"`
	Sz        string    `xml:"sz>val"`
}

type docxHyperlink struct {
	RID  string        `xml:"id,attr"`
	Runs []docxRunFull `xml:"r"`
}

type docxTable struct {
	Rows []docxTableRow `xml:"tr"`
}

type docxTableRow struct {
	Cells []docxTableCell `xml:"tc"`
}

type docxTableCell struct {
	Paragraphs []docxParagraphFull `xml:"p"`
}

type docxNumbering struct {
	AbstractNums []docxAbstractNum `xml:"abstractNum"`
	Nums         []docxNum         `xml:"num"`
}

type docxAbstractNum struct {
	AbstractNumID string    `xml:"abstractNumId,attr"`
	Lvls          []docxLvl `xml:"lvl"`
}

type docxLvl struct {
	Ilvl    string `xml:"ilvl,attr"`
	NumFmt  string `xml:"numFmt>val"`
	LvlText string `xml:"lvlText>val"`
}

type docxNum struct {
	NumID         string `xml:"numId,attr"`
	AbstractNumID string `xml:"abstractNumId>val"`
}

type docxRelationships struct {
	Relationships []docxRelationship `xml:"Relationship"`
}

type docxRelationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
	Type   string `xml:"Type,attr"`
}

func (m *Markitdown) convertDocx(data []byte) (*Result, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open docx: %w", err)
	}

	files := make(map[string][]byte)
	for _, f := range reader.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		// Store under both raw and normalized path
		files[f.Name] = b
		norm := f.Name
		if len(norm) > 2 && norm[:2] == "./" {
			norm = norm[2:]
			files[norm] = b
		}
	}

	docXML, ok := files["word/document.xml"]
	if !ok {
		return nil, fmt.Errorf("word/document.xml not found")
	}

	docXML = stripDocxNS(docXML)

	stylesXML := files["word/styles.xml"]
	if len(stylesXML) > 0 {
		stylesXML = stripDocxNS(stylesXML)
	}

	relsXML := files["word/_rels/document.xml.rels"]

	numberingXML := files["word/numbering.xml"]
	if len(numberingXML) > 0 {
		numberingXML = stripDocxNS(numberingXML)
	}

	doc := m.parseDocxDocument(docXML)
	styles := m.parseDocxStyles(stylesXML)
	rels := m.parseDocxRelationships(relsXML)
	numbering := m.parseDocxNumbering(numberingXML)

	var md strings.Builder
	m.writeDocxBody(&md, doc, styles, rels, numbering)

	plainText := stripMarkdown(md.String())

	return &Result{
		Content:     md.String(),
		TextContent: plainText,
	}, nil
}

func (m *Markitdown) parseDocxDocument(data []byte) docxDocument {
	var doc docxDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		mdLog.Warnf("Failed to parse docx XML: %v", err)
	}
	return doc
}

func (m *Markitdown) parseDocxStyles(data []byte) map[string]int {
	headingStyles := make(map[string]int)
	if len(data) == 0 {
		return headingStyles
	}

	var styles docxStyles
	if err := xml.Unmarshal(data, &styles); err != nil {
		mdLog.Warnf("Failed to parse styles: %v", err)
		return headingStyles
	}

	for _, s := range styles.Styles {
		if s.Type == "paragraph" && s.OutlineLvl >= 0 && s.OutlineLvl < 9 {
			headingStyles[s.StyleID] = s.OutlineLvl + 1
		}
	}
	return headingStyles
}

func (m *Markitdown) parseDocxRelationships(data []byte) map[string]string {
	rels := make(map[string]string)
	if len(data) == 0 {
		return rels
	}
	var r docxRelationships
	if err := xml.Unmarshal(data, &r); err != nil {
		return rels
	}
	for _, rel := range r.Relationships {
		rels[rel.ID] = rel.Target
	}
	return rels
}

func (m *Markitdown) parseDocxNumbering(data []byte) map[string]string {
	numFmts := make(map[string]string)
	if len(data) == 0 {
		return numFmts
	}
	var n docxNumbering
	if err := xml.Unmarshal(data, &n); err != nil {
		return numFmts
	}
	abstractFmts := make(map[string]string)
	for _, an := range n.AbstractNums {
		for _, lvl := range an.Lvls {
			if lvl.Ilvl == "0" {
				abstractFmts[an.AbstractNumID] = lvl.NumFmt
				break
			}
		}
	}
	for _, num := range n.Nums {
		if fmt, ok := abstractFmts[num.AbstractNumID]; ok {
			numFmts[num.NumID] = fmt
		}
	}
	return numFmts
}

func (m *Markitdown) writeDocxBody(md *strings.Builder, doc docxDocument, headingStyles map[string]int, rels map[string]string, numFmts map[string]string) {
	elements := make([]interface{}, 0)
	for _, p := range doc.Body.Paragraphs {
		elements = append(elements, p)
	}
	for _, t := range doc.Body.Tables {
		elements = append(elements, t)
	}
	for _, el := range elements {
		switch v := el.(type) {
		case docxParagraphFull:
			m.writeDocxParagraph(md, v, headingStyles, rels, numFmts)
		case docxTable:
			m.writeDocxTable(md, v, headingStyles, rels, numFmts)
		}
	}
}

func (m *Markitdown) writeDocxParagraph(md *strings.Builder, p docxParagraphFull, headingStyles map[string]int, rels map[string]string, numFmts map[string]string) {
	text := m.extractParagraphText(p, rels)
	if text == "" {
		md.WriteString("\n")
		return
	}

	if p.PPr != nil && p.PPr.StyleID != "" {
		if level, ok := headingStyles[p.PPr.StyleID]; ok {
			md.WriteString(strings.Repeat("#", level) + " " + text + "\n\n")
			return
		}
	}

	if p.PPr != nil && p.PPr.NumPr != nil {
		numID := p.PPr.NumPr.NumID
		ilvl := p.PPr.NumPr.Ilvl
		indent := ""
		if lvl, err := strconv.Atoi(ilvl); err == nil && lvl > 0 {
			indent = strings.Repeat("  ", lvl)
		}
		if fmt, ok := numFmts[numID]; ok && fmt == "bullet" {
			md.WriteString(indent + "- " + text + "\n")
		} else {
			md.WriteString(indent + "1. " + text + "\n")
		}
		return
	}

	md.WriteString(text + "\n\n")
}

func (m *Markitdown) extractParagraphText(p docxParagraphFull, rels map[string]string) string {
	var parts []string

	for _, r := range p.Runs {
		t := strings.TrimSpace(r.Text)
		if t == "" {
			continue
		}
		t = m.applyRunFormatting(t, r.RPr)
		parts = append(parts, t)
	}

	for _, h := range p.Hyperlinks {
		var linkTexts []string
		for _, r := range h.Runs {
			t := strings.TrimSpace(r.Text)
			if t == "" {
				continue
			}
			t = m.applyRunFormatting(t, r.RPr)
			linkTexts = append(linkTexts, t)
		}
		linkText := strings.Join(linkTexts, "")
		if linkText != "" {
			if target, ok := rels[h.RID]; ok {
				parts = append(parts, fmt.Sprintf("[%s](%s)", linkText, target))
			} else {
				parts = append(parts, linkText)
			}
		}
	}

	return strings.Join(parts, " ")
}

func (m *Markitdown) applyRunFormatting(text string, rPr *docxRPr) string {
	if rPr == nil {
		return text
	}
	isBold := rPr.Bold != nil || rPr.BoldVal == "1" || rPr.BoldVal == "true"
	isItalic := rPr.Italic != nil || rPr.ItalicVal == "1" || rPr.ItalicVal == "true"

	if isBold && isItalic {
		return "***" + text + "***"
	}
	if isBold {
		return "**" + text + "**"
	}
	if isItalic {
		return "*" + text + "*"
	}
	return text
}

func (m *Markitdown) writeDocxTable(md *strings.Builder, table docxTable, _ map[string]int, rels map[string]string, numFmts map[string]string) {
	if len(table.Rows) == 0 {
		return
	}

	maxCols := 0
	for _, row := range table.Rows {
		if len(row.Cells) > maxCols {
			maxCols = len(row.Cells)
		}
	}
	if maxCols == 0 {
		return
	}

	m.writeTableRow(md, table.Rows[0], maxCols, rels, numFmts)
	md.WriteString("|" + strings.Repeat(" --- |", maxCols) + "\n")
	for i := 1; i < len(table.Rows); i++ {
		m.writeTableRow(md, table.Rows[i], maxCols, rels, numFmts)
	}
	md.WriteString("\n")
}

func (m *Markitdown) writeTableRow(md *strings.Builder, row docxTableRow, maxCols int, rels, _ map[string]string) {
	md.WriteString("|")
	for i, cell := range row.Cells {
		if i >= maxCols {
			break
		}
		var cellTexts []string
		for _, p := range cell.Paragraphs {
			t := m.extractParagraphText(p, rels)
			if t != "" {
				cellTexts = append(cellTexts, t)
			}
		}
		md.WriteString(" " + strings.Join(cellTexts, " ") + " |")
	}
	for i := len(row.Cells); i < maxCols; i++ {
		md.WriteString("  |")
	}
	md.WriteString("\n")
}

// --- DOC Conversion ---

func (m *Markitdown) convertDoc(data []byte) (*Result, error) {
	if isOLE2(data) {
		text, err := parseOLE2WordDocRaw(data)
		if err != nil {
			return nil, err
		}
		return &Result{Content: text, TextContent: text}, nil
	}

	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err == nil {
		return m.convertDocx(data)
	}

	text := string(data)
	return &Result{Content: text, TextContent: text}, nil
}

func parseOLE2WordDocRaw(data []byte) (string, error) {
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

	// Try piece-table-based extraction first
	if text, ok := extractDocViaPieceTable(wordStream, tableStream); ok {
		return text, nil
	}

	// Fallback: skip FIB header + UTF-16LE scan (WordDocument only)
	const fibHeaderSize = 1024
	textStart := wordStream
	if len(textStart) > fibHeaderSize {
		textStart = textStart[fibHeaderSize:]
	}

	text := tryDecodeUTF16LE(textStart)
	if len(text) > 20 {
		return text, nil
	}

	if len(textStart) > 0 {
		if decoded, err := tryDecodeGBK(textStart); err == nil && len(decoded) > 20 {
			return decoded, nil
		}
	}

	return extractPrintableStrings(wordStream), nil
}

// --- PDF Conversion ---

func (m *Markitdown) convertPDF(data []byte) (*Result, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}

	textReader, err := reader.GetPlainText()
	if err != nil {
		return nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	text, err := io.ReadAll(textReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF text: %w", err)
	}

	md := string(text)
	return &Result{Content: md, TextContent: md}, nil
}

// stripMarkdown removes Markdown formatting for plain text extraction.
func stripMarkdown(md string) string {
	var result strings.Builder
	lines := strings.Split(md, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			text := strings.TrimLeft(trimmed, "# ")
			result.WriteString(text + "\n")
		} else if strings.HasPrefix(trimmed, "|") {
			if strings.Contains(trimmed, "---") {
				continue
			}
			cells := strings.Split(trimmed, "|")
			var cellTexts []string
			for _, c := range cells {
				t := strings.TrimSpace(c)
				if t != "" {
					cellTexts = append(cellTexts, t)
				}
			}
			result.WriteString(strings.Join(cellTexts, "\t") + "\n")
		} else if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "1. ") || strings.HasPrefix(trimmed, "* ") {
			result.WriteString(trimmed + "\n")
		} else {
			result.WriteString(line + "\n")
		}
	}
	plain := strings.ReplaceAll(result.String(), "**", "")
	plain = strings.ReplaceAll(plain, "***", "")
	plain = strings.ReplaceAll(plain, "__", "")
	return plain
}
