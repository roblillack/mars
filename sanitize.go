package mars

import "regexp"

var lineBreakPattern = regexp.MustCompile(`[\r\n]+`)

func removeLineBreaks(s string) string {
	return lineBreakPattern.ReplaceAllString(s, " ")
}

func removeAllWhitespace(s string) string {
	return whiteSpacePattern.ReplaceAllString(s, "")
}
