package mars

import (
	"bytes"
	"html/template"
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

func TestTemplateFuncs(t *testing.T) {
	type Scenario struct {
		T string
		D Args
		R string
		E string
	}
	for _, scenario := range []Scenario{
		{
			`<a href="/{{slug .title}}">{{.title}}</a>`,
			Args{"title": "This is a Blog Post!"},
			`<a href="/this-is-a-blog-post">This is a Blog Post!</a>`,
			``,
		},
		{
			`{{raw .title}}`,
			Args{"title": "<b>bla</b>"},
			`<b>bla</b>`,
			``,
		},
		{
			`{{if even .no}}yes{{else}}no{{end}}`,
			Args{"no": 0},
			`yes`,
			``,
		},
		{
			`{{if even .no}}yes{{else}}no{{end}}`,
			Args{"no": 1},
			`no`,
			``,
		},
	} {
		tmpl, err := template.New("foo").Funcs(TemplateFuncs).Parse(scenario.T)
		if err != nil {
			t.Error(err)
		}
		buf := &strings.Builder{}
		err = goTemplateWrapper{loader: nil, funcMap: nil, Template: tmpl}.Template.Execute(buf, scenario.D)
		if err != nil {
			t.Error(err)
		}
		if res := buf.String(); res != scenario.R {
			t.Errorf("Expected '%s', got '%s' for input '%s'", scenario.R, res, scenario.T)
		}
	}
}

func TestTemplateParsingErrors(t *testing.T) {
	for _, scenario := range []string{
		`{{.uhoh}`,
		`{{if .condition}}look{{else}}there's no end here`,
		`{{undefined_function .parameter}}`,
	} {
		_, err := template.New("foo").Funcs(TemplateFuncs).Parse(scenario)
		if err == nil {
			t.Errorf("No error when parsing: %s", scenario)
		}
	}
}
