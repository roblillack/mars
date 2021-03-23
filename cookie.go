package mars

import (
	"net/url"
	"regexp"
)

var (
	cookieKeyValueParser = regexp.MustCompile("\x00([^:]*):([^\x00]*)\x00")
)

// parseKeyValueCookie takes the raw (escaped) cookie value and parses out key values.
func parseKeyValueCookie(val string, cb func(key, val string)) {
	val, _ = url.QueryUnescape(val)
	if matches := cookieKeyValueParser.FindAllStringSubmatch(val, -1); matches != nil {
		for _, match := range matches {
			cb(match[1], match[2])
		}
	}
}
