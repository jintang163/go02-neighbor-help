package validate

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func Trim(s string) string { return strings.TrimSpace(s) }

func RuneLen(s string) int { return utf8.RuneCountInString(s) }

func InRange(s string, min, max int) bool {
	n := RuneLen(strings.TrimSpace(s))
	return n >= min && n <= max
}

func SanitizePlain(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsControl(r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func ContainsFold(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
