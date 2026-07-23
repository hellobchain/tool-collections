package services

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// DocCleaner handles cleaning operations on Word documents such as
// removing comments and accepting track changes.
type DocCleaner struct{}

// CleanOptions defines which cleaning operations to perform.
type CleanOptions struct {
	RemoveComments bool // Remove all comments from the document
	AcceptChanges  bool // Accept all tracked changes (insertions/deletions)
}

// defaultCleanOptions returns options with all cleaning enabled.
func defaultCleanOptions() CleanOptions {
	return CleanOptions{
		RemoveComments: true,
		AcceptChanges:  true,
	}
}

// CleanDocx cleans a DOCX file according to the given options.
// Returns the cleaned DOCX bytes.
func (c *DocCleaner) CleanDocx(data []byte, opts CleanOptions) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open docx: %w", err)
	}

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	removeComments := opts.RemoveComments
	acceptChanges := opts.AcceptChanges

	for _, f := range reader.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, _ := io.ReadAll(rc)
		rc.Close()

		// Skip comment files entirely if removing comments
		if removeComments && isCommentFile(f.Name) {
			continue
		}

		// Process document XML for comment markers and track changes
		if f.Name == "word/document.xml" {
			content = c.cleanDocumentXML(content, removeComments, acceptChanges)
		}

		// Process header/footer files too
		if (strings.HasPrefix(f.Name, "word/header") || strings.HasPrefix(f.Name, "word/footer")) &&
			strings.HasSuffix(f.Name, ".xml") {
			content = c.cleanDocumentXML(content, removeComments, acceptChanges)
		}

		wc, err := writer.Create(f.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create entry %s: %w", f.Name, err)
		}
		wc.Write(content)
	}

	writer.Close()
	return buf.Bytes(), nil
}

func isCommentFile(name string) bool {
	return name == "word/comments.xml" ||
		name == "word/commentsExtensible.xml" ||
		strings.HasPrefix(name, "word/comments")
}

// cleanDocumentXML removes comment markers and/or accepts track changes.
// Works on raw XML (with w: namespace) to produce valid output.
func (c *DocCleaner) cleanDocumentXML(content []byte, removeComments, acceptChanges bool) []byte {
	s := string(content)

	if removeComments {
		s = removeCommentMarkers(s)
	}

	if acceptChanges {
		s = acceptTrackChanges(s)
	}

	return []byte(s)
}

// removeCommentMarkers strips comment range markers and reference elements.
func removeCommentMarkers(s string) string {
	// Remove <w:commentRangeStart w:id="..."/> and <w:commentRangeEnd w:id="..."/>
	s = commentRangeReplacer.Replace(s)
	// Remove <w:commentReference w:id="..."/>
	s = commentRefReplacer.Replace(s)

	return s
}

var commentRangeReplacer = mustNewReplacer(
	`<w:commentRangeStart`, `/>`,
	`<w:commentRangeEnd`, `/>`,
)

var commentRefReplacer = mustNewReplacer(
	`<w:commentReference`, `/>`,
)

// acceptTrackChanges accepts all tracked changes:
// - <w:ins>...</w:ins> → keep content, remove wrapper
// - <w:del>...</w:del> → remove entirely (content + wrapper)
func acceptTrackChanges(s string) string {
	// Process deletions first: remove <w:del>...</w:del> entirely
	s = removeDelElements(s)
	// Process insertions: unwrap <w:ins>...</w:ins> → keep content only
	s = unwrapInsElements(s)
	return s
}

// removeDelElements removes all <w:del ...>...</w:del> blocks (including content).
func removeDelElements(s string) string {
	var result strings.Builder
	for {
		start := strings.Index(s, "<w:del")
		if start < 0 {
			result.WriteString(s)
			break
		}
		result.WriteString(s[:start])

		// Find the end of the opening tag
		rest := s[start:]
		tagEnd := strings.IndexByte(rest, '>')
		if tagEnd < 0 {
			result.WriteString(rest)
			break
		}

		// Check if it's a self-closing tag
		if tagEnd > 0 && rest[tagEnd-1] == '/' {
			s = s[start+tagEnd+1:]
			continue
		}

		// Find matching </w:del>
		depth := 1
		pos := tagEnd + 1
		for depth > 0 && pos < len(rest) {
			nextOpen := strings.Index(rest[pos:], "<w:del")
			nextClose := strings.Index(rest[pos:], "</w:del>")
			if nextClose < 0 {
				// No proper closing tag; skip to end
				pos = len(rest)
				break
			}
			if nextOpen >= 0 && nextOpen < nextClose {
				depth++
				pos += nextOpen + len("<w:del")
			} else {
				depth--
				pos += nextClose + len("</w:del>")
			}
		}
		s = s[start+pos:]
	}
	return result.String()
}

// unwrapInsElements removes <w:ins> and </w:ins> tags, keeping the content.
func unwrapInsElements(s string) string {
	// Remove opening <w:ins ...> tags (including any attributes)
	re := regexp.MustCompile(`<w:ins[^>]*>`)
	s = re.ReplaceAllString(s, "")
	// Remove closing </w:ins> tags
	s = strings.ReplaceAll(s, "</w:ins>", "")
	return s
}

// mustNewReplacer creates a tag-stripping replacer that removes XML elements
// matching the given prefix and ending pattern.
type tagStripper struct {
	prefixes []string
	suffix   string
}

func mustNewReplacer(prefixes ...string) *tagStripper {
	if len(prefixes) == 0 {
		panic("at least one prefix required")
	}
	suffix := prefixes[len(prefixes)-1]
	prefixes = prefixes[:len(prefixes)-1]
	return &tagStripper{prefixes: prefixes, suffix: suffix}
}

func (ts *tagStripper) Replace(s string) string {
	for _, prefix := range ts.prefixes {
		s = ts.removeTags(s, prefix, ts.suffix)
	}
	return s
}

func (ts *tagStripper) removeTags(s, openPrefix, closeSuffix string) string {
	var result strings.Builder
	for {
		idx := strings.Index(s, openPrefix)
		if idx < 0 {
			result.WriteString(s)
			break
		}
		result.WriteString(s[:idx])
		rest := s[idx:]
		tagEnd := strings.IndexByte(rest, '>')
		if tagEnd < 0 {
			result.WriteString(rest)
			break
		}
		s = s[idx+tagEnd+1:]
	}
	return result.String()
}

// CleanDoc cleans a .doc file by first converting to DOCX-like processing.
// For .doc files, we can only do basic text-level cleaning since the
// binary format doesn't support surgical comment/changes removal.
func (c *DocCleaner) CleanDoc(data []byte, opts CleanOptions) ([]byte, error) {
	// For .doc files, extract text via OLE2, clean it, and return as-is
	// (we can't easily reconstruct a .doc file with comments removed)
	text, err := ReadDoc(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read doc: %w", err)
	}

	// Comments and track changes are already stripped in the text extraction
	// (our piece-table parser only extracts visible text)
	return []byte(text), nil
}

// CleanFile detects the file type and applies the appropriate cleaning.
// For .docx, it returns cleaned DOCX bytes. For .doc/.txt, it returns plain text.
func (c *DocCleaner) CleanFile(filename string, data []byte, opts CleanOptions) ([]byte, error) {
	ext := strings.ToLower(filename[strings.LastIndexByte(filename, '.'):])
	switch ext {
	case ".docx":
		return c.CleanDocx(data, opts)
	case ".doc":
		return c.CleanDoc(data, opts)
	case ".txt":
		return data, nil
	default:
		return data, nil
	}
}
