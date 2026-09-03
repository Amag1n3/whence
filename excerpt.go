package main

import "strings"

// excerptWidth is the display width used when whence prints a reasoning
// excerpt alongside an anchor.
const excerptWidth = 72

// Excerpt shortens a captured reasoning span for single-line display. It
// collapses internal whitespace and appends an ellipsis when the span had to
// be cut short.
func Excerpt(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) < max {
		return s
	}
	return s[:max-3] + "..."
}
