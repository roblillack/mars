package mars

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestContextAwareRenderFuncs(t *testing.T) {
	loadMessages(testDataPath)

	_, filename, _, _ := runtime.Caller(0)
	BasePath = filepath.Join(filepath.Dir(filename), "testdata")
	SetupViews()

	for expected, input := range map[string]interface{}{
		"<h1>Hey, there <b>Rob</b>!</h1>":   "Rob",
		"<h1>Hey, there <b>&lt;3</b>!</h1>": Blarp("<3"),
	} {
		result := runRequest("en", "i18n_ctx.html", Args{"input": input})
		if result != expected {
			t.Errorf("Expected '%s', got '%s' for input '%s'", expected, result, input)
		}
	}
}
