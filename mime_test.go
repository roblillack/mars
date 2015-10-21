package mars

import (
	"testing"
)

func TestContentTypeByFilename(t *testing.T) {
	testCases := map[string]string{
		"xyz.jpg":       "image/jpeg",
		"helloworld.c":  "text/x-c; charset=utf-8",
		"helloworld.":   "application/octet-stream",
		"helloworld":    "application/octet-stream",
		"hello.world.c": "text/x-c; charset=utf-8",
	}
	for filename, expected := range testCases {
		actual := ContentTypeByFilename(filename)
		if actual != expected {
			t.Errorf("%s: %s, Expected %s", filename, actual, expected)
		}
	}
}

func TestCustomMimeTypes(t *testing.T) {
	startFakeBookingApp()

	if ct := ContentTypeByFilename("B1F1AA4C-8156-4649-9248-0DE19BD63164.bkng"); ct != "application/x-booking" {
		t.Errorf("Wrong MIME type returned: %s", t)
	}
}
