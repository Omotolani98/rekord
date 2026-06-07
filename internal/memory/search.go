package memory

import "strings"

func scoreMemory(m Memory, query string) int {
	score := 0
	if strings.Contains(strings.ToLower(m.Title), query) {
		score += 8
	}
	if strings.Contains(strings.ToLower(m.Body), query) {
		score += 5
	}
	for _, tag := range m.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			score += 10
		}
	}
	for _, file := range m.RelatedFiles {
		if strings.Contains(strings.ToLower(file), query) {
			score += 7
		}
	}
	if strings.Contains(strings.ToLower(m.Agent), query) {
		score += 4
	}
	if strings.Contains(strings.ToLower(m.SessionName), query) || strings.Contains(strings.ToLower(m.SessionID), query) {
		score += 4
	}
	if m.Status == StatusOpen {
		score++
	}
	return score
}
