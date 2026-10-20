package contenttags

import (
	"golang.org/x/text/unicode/norm"
	"strings"
)

const AIGC = "AIGC"

func isAIGC(tag string) bool {
	return strings.EqualFold(strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(norm.NFKC.String(tag)), "#")), AIGC)
}

func HasAIGC(tags []string) bool {
	for _, tag := range tags {
		if isAIGC(tag) {
			return true
		}
	}
	return false
}

func WithAIGC(tags []string, declared bool) []string {
	out := make([]string, 0, len(tags)+1)
	seen := false
	for _, tag := range tags {
		if isAIGC(tag) {
			if !seen {
				out = append(out, AIGC)
				seen = true
			}
			continue
		}
		out = append(out, tag)
	}
	if declared && !seen {
		out = append(out, AIGC)
	}
	return out
}

func MergeGenerated(generated, declared []string) []string {
	out := make([]string, 0, len(generated)+1)
	for _, tag := range generated {
		if !isAIGC(tag) {
			out = append(out, tag)
		}
	}
	return WithAIGC(out, HasAIGC(declared))
}
