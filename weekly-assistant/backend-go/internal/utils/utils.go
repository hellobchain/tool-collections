package utils

import (
	"net/url"
	"strings"
)

func PercentEncode(s string) string {
	encoded := url.QueryEscape(s)
	return strings.ReplaceAll(encoded, "+", "%20")
}
