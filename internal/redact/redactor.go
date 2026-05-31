package redact

import (
	"regexp"
	"sort"
)

const Placeholder = "[REDACTED]"

type Redactor struct {
	patterns []Pattern
}

func New(patterns []Pattern) *Redactor {
	return &Redactor{patterns: patterns}
}

func NewDefault() *Redactor {
	return New(DefaultPatterns())
}

func Custom(category string, re *regexp.Regexp) Pattern {
	return Pattern{Category: category, Re: re, Replacement: Placeholder}
}

func (r *Redactor) Redact(s string) string {
	for _, p := range r.patterns {
		s = p.Re.ReplaceAllString(s, p.Replacement)
	}
	return s
}

func (r *Redactor) Scan(s string) []string {
	seen := make(map[string]struct{})
	for _, p := range r.patterns {
		if p.Re.MatchString(s) {
			seen[p.Category] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}
