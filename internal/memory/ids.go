package memory

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
	"unicode"
)

func NewID(prefix, title string, now time.Time) string {
	stamp := now.UTC().Format("20060102-150405")
	slug := slug(title)
	if slug == "" {
		slug = prefix
	}
	return prefix + "_" + stamp + "_" + slug + "_" + randomSuffix()
}

func slug(s string) string {
	var b strings.Builder
	prevDash := true
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) && r < 128:
			b.WriteRune(r)
			prevDash = false
		case unicode.IsDigit(r) && r < 128:
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
		if b.Len() >= 48 {
			break
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func randomSuffix() string {
	var b [3]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "000000"
	}
	return hex.EncodeToString(b[:])
}
