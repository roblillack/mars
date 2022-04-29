package mars

import (
	"testing"
)

func TestRemovingLineBreaks(t *testing.T) {
	for i, exp := range map[string]string{
		"This is a test.":             "This is a test.",
		"This is\n a test.":           "This is  a test.",
		"This is\r a test.":           "This is  a test.",
		"This is\r\n a test.":         "This is  a test.",
		"\n\n\n\n\nThis is\r a test.": " This is  a test.",
	} {
		if res := removeLineBreaks(i); res != exp {
			t.Errorf("Unexpected result '%s' when removing line breaks from '%s'.\n", res, i)
		}
	}
}

func TestRemovingAllWhitespace(t *testing.T) {
	for i, exp := range map[string]string{
		"This is a test.":             "Thisisatest.",
		"This is\n a test.":           "Thisisatest.",
		"This is\r a test.":           "Thisisatest.",
		"This is\r\n a test.":         "Thisisatest.",
		"\n\n\n\n\nThis is\r a test.": "Thisisatest.",
	} {
		if res := removeAllWhitespace(i); res != exp {
			t.Errorf("Unexpected result '%s' when removing all whitespace from '%s'.\n", res, i)
		}
	}
}
