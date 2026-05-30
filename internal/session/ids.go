package session

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

const fallbackSlug = "session"

func NewID(name string, now time.Time) string {
	stamp := now.UTC().Format("20060102-150405")
	return stamp + "-" + slug(name)
}

func slug(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	prevDash := true
	for _, r := range strings.ToLower(name) {
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
	}
	out := strings.TrimRight(b.String(), "-")
	if out == "" {
		return fallbackSlug
	}
	return out
}

func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}
	return nil
}
