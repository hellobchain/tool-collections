package utils

import (
	"net/url"
	"strings"
)

func PercentEncode(s string) string {
	encoded := url.QueryEscape(s)
	return strings.ReplaceAll(encoded, "+", "%20")
}

func IsTextRune(r rune) bool {
	switch {
	case r >= 0x20 && r <= 0x7E:
		return true
	case r >= 0x4E00 && r <= 0x9FFF:
		return true
	case r >= 0x3400 && r <= 0x4DBF:
		return true
	case r >= 0x2E80 && r <= 0x2EFF:
		return true
	case r >= 0x3000 && r <= 0x303F:
		return true
	case r >= 0xFF00 && r <= 0xFFEF:
		return true
	case r >= 0x2000 && r <= 0x206F:
		return true
	case r >= 0xFE30 && r <= 0xFE4F:
		return true
	case r >= 0x00A0 && r <= 0x00FF:
		return true
	case r >= 0x0100 && r <= 0x024F:
		return true
	case r >= 0x0370 && r <= 0x03FF:
		return true
	case r >= 0x0400 && r <= 0x04FF:
		return true
	case r == 0x0A || r == 0x0D:
		return true
	default:
		return false
	}
}
