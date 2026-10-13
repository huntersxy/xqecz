// Package contenttags owns user-declared content tag markers shared by upload and generated metadata.
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

// WithAIGC preserves manual tags and canonicalizes the marker once. Unchecking does not erase a manually supplied marker.
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

// MergeGenerated treats AIGC as a user declaration, never a model inference. The marker is outside the generated tag limit.
func MergeGenerated(generated, declared []string) []string {
	out := make([]string, 0, len(generated)+1)
	for _, tag := range generated {
		if !isAIGC(tag) {
			out = append(out, tag)
		}
	}
	return WithAIGC(out, HasAIGC(declared))
}
