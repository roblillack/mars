package mars

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func setupTemplateTestingApp() {
	_, filename, _, _ := runtime.Caller(0)
	BasePath = filepath.Join(filepath.Dir(filename), "testdata")
	SetupViews()
}

func TestContextAwareRenderFuncs(t *testing.T) {
	setupTemplateTestingApp()
	loadMessages(testDataPath)

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

func simulateRequest(format, view string) string {
	w := httptest.NewRecorder()
	httpRequest, _ := http.NewRequest("GET", "/", nil)
	req := NewRequest(httpRequest)
	req.Format = format
	c := NewController(req, &Response{Out: w})
	c.RenderTemplate(view).Apply(c.Request, c.Response)

	buf := &bytes.Buffer{}
	buf.ReadFrom(w.Body)
	return buf.String()
}

func TestTemplateNotAvailable(t *testing.T) {
	setupTemplateTestingApp()
	expectedString := "Template non_existant.html not found."

	if resp := simulateRequest("html", "non_existant.html"); !strings.Contains(resp, expectedString) {
		t.Error("Error rendering template error message for plaintext requests. Got:", resp)
	}
	if resp := simulateRequest("txt", "non_existant.html"); !strings.Contains(resp, expectedString) {
		t.Error("Error rendering template error message for plaintext requests. Got:", resp)
	}
}
