package store

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode"

	"go02-neighbor-help/internal/validate"
)

func defaultIDGenerator(prefix string) string {
	b := make([]byte, 9)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s%d%d", prefix, time.Now().UnixNano(), len(prefix))
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b)
}

func sanitizeDisplayName(s string) string {
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
	return validate.Trim(b.String())
}

func matchQuery(haystack, needle string) bool {
	return validate.ContainsFold(haystack, needle)
}
