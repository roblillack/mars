package mars

//go:generate go-bindata -pkg $GOPACKAGE -prefix templates -o embedded_templates.go templates/errors/

import (
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ERROR_CLASS = "hasError"

// This object handles loading and parsing of templates.
// Everything below the application's views directory is treated as a template.
type TemplateLoader struct {
	// This is the set of all templates under views
	templateSet *template.Template
	// If an error was encountered parsing the templates, it is stored here.
	compileError *Error
	// Paths to search for templates, in priority order.
	paths []string
	// Map from template name to the path from whence it was loaded.
	templatePaths map[string]string
	// templateNames is a map from lower case template name to the real template name.
	templateNames map[string]string
}

type Template interface {
	Name() string
	Content() []string
	Render(wr io.Writer, arg interface{}) error
}

var invalidSlugPattern = regexp.MustCompile(`[^a-z0-9 _-]`)
var whiteSpacePattern = regexp.MustCompile(`\s+`)

var (
	// The functions available for use in the templates.
	TemplateFuncs = map[string]interface{}{
		"url": ReverseUrl,
		"set": func(renderArgs map[string]interface{}, key string, value interface{}) template.JS {
			renderArgs[key] = value
			return template.JS("")
		},
		"append": func(renderArgs map[string]interface{}, key string, value interface{}) template.JS {
			if renderArgs[key] == nil {
				renderArgs[key] = []interface{}{value}
			} else {
				renderArgs[key] = append(renderArgs[key].([]interface{}), value)
			}
			return template.JS("")
		},
		"field": NewField,
		"firstof": func(args ...interface{}) interface{} {
			for _, val := range args {
				switch val.(type) {
				case nil:
					continue
				case string:
					if val == "" {
						continue
					}
					return val
				default:
					return val
				}
			}
			return nil
		},
		"option": func(f *Field, val interface{}, label string) template.HTML {
			selected := ""
			if f.Flash() == val || (f.Flash() == "" && f.Value() == val) {
				selected = " selected"
			}

			return template.HTML(fmt.Sprintf(`<option value="%s"%s>%s</option>`,
				html.EscapeString(fmt.Sprintf("%v", val)), selected, html.EscapeString(label)))
		},
		"radio": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Flash() == val {
				checked = " checked"
			}
			return template.HTML(fmt.Sprintf(`<input type="radio" name="%s" value="%s"%s>`,
				html.EscapeString(f.Name), html.EscapeString(val), checked))
		},
		"checkbox": func(f *Field, val string) template.HTML {
			checked := ""
			if f.Flash() == val {
				checked = " checked"
			}
			return template.HTML(fmt.Sprintf(`<input type="checkbox" name="%s" value="%s"%s>`,
				html.EscapeString(f.Name), html.EscapeString(val), checked))
		},
		// Pads the given string with &nbsp;'s up to the given width.
		"pad": func(str string, width int) template.HTML {
			if len(str) >= width {
				return template.HTML(html.EscapeString(str))
			}
			return template.HTML(html.EscapeString(str) + strings.Repeat("&nbsp;", width-len(str)))
		},

		"errorClass": func(name string, renderArgs map[string]interface{}) template.HTML {
			errorMap, ok := renderArgs["errors"].(map[string]*ValidationError)
			if !ok || errorMap == nil {
				WARN.Println("Called 'errorClass' without 'errors' in the render args.")
				return template.HTML("")
			}
			valError, ok := errorMap[name]
			if !ok || valError == nil {
				return template.HTML("")
			}
			return template.HTML(ERROR_CLASS)
		},

		"msg": func(renderArgs map[string]interface{}, message string, args ...interface{}) template.HTML {
			str, ok := renderArgs[CurrentLocaleRenderArg].(string)
			if !ok {
				return template.HTML(``)
			}
			return MessageHTML(str, message, args...)
		},

		// Dummy function to tell the allow for signature checking when compiling the templates
		"t": func(message string, args ...interface{}) template.HTML { return template.HTML(``) },

		// Replaces newlines with <br>
		"nl2br": func(text string) template.HTML {
			return template.HTML(strings.Replace(template.HTMLEscapeString(text), "\n", "<br>", -1))
		},

		// Skips sanitation on the parameter.  Do not use with dynamic data.
		"raw": func(text string) template.HTML {
			return template.HTML(text)
		},

		// Pluralize, a helper for pluralizing words to correspond to data of dynamic length.
		// items - a slice of items, or an integer indicating how many items there are.
		// pluralOverrides - optional arguments specifying the output in the
		//     singular and plural cases.  by default "" and "s"
		"pluralize": func(items interface{}, pluralOverrides ...string) string {
			singular, plural := "", "s"
			if len(pluralOverrides) >= 1 {
				singular = pluralOverrides[0]
				if len(pluralOverrides) == 2 {
					plural = pluralOverrides[1]
				}
			}

			switch v := reflect.ValueOf(items); v.Kind() {
			case reflect.Int:
				if items.(int) != 1 {
					return plural
				}
			case reflect.Slice:
				if v.Len() != 1 {
					return plural
				}
			default:
				ERROR.Println("pluralize: unexpected type: ", v)
			}
			return singular
		},

		// Format a date according to the application's default date(time) format.
		"date": func(date time.Time) string {
			return date.Format(DateFormat)
		},
		"datetime": func(date time.Time) string {
			return date.Format(DateTimeFormat)
		},
		"slug": Slug,
		"even": func(a int) bool { return (a % 2) == 0 },
	}
)

func NewTemplateLoader(paths []string) *TemplateLoader {
	loader := &TemplateLoader{
		paths: paths,
	}
	return loader
}

// emptyTemplateLoader creates an empty TemplateLoader that will only ever support the embedded Mars templates
// for returning results.
func emptyTemplateLoader() *TemplateLoader {
	t := &TemplateLoader{}
	t.Refresh()
	return t
}

func (loader *TemplateLoader) createEmptyTemplateSet() *Error {
	// Create the template set.  This panics if any of the funcs do not
	// conform to expectations, so we wrap it in a func and handle those
	// panics by serving an error page.
	var funcError *Error
	func() {
		defer func() {
			if err := recover(); err != nil {
				funcError = &Error{
					Title:       "Panic (Template Loader)",
					Description: fmt.Sprintln(err),
				}
			}
		}()
		loader.templateSet = template.New("_").Funcs(TemplateFuncs)
		loader.templateSet.Parse("")
	}()

	return funcError
}

// This scans the views directory and parses all templates as Go Templates.
// If a template fails to parse, the error is set on the loader.
// (It's awkward to refresh a single Go Template)
func (loader *TemplateLoader) Refresh() error {
	TRACE.Printf("Refreshing templates from %s", loader.paths)

	loader.compileError = nil
	loader.templatePaths = map[string]string{}
	loader.templateNames = map[string]string{}

	if err := loader.createEmptyTemplateSet(); err != nil {
		return err
	}

	// Walk through the template loader's paths and build up a template set.
	for _, basePath := range loader.paths {
		// Walk only returns an error if the template loader is completely unusable
		// (namely, if one of the TemplateFuncs does not have an acceptable signature).

		// Handling symlinked directories
		var fullSrcDir string
		f, err := os.Lstat(basePath)
		if err == nil && f.Mode()&os.ModeSymlink == os.ModeSymlink {
			fullSrcDir, err = filepath.EvalSymlinks(basePath)
			if err != nil {
				panic(err)
			}
		} else {
			fullSrcDir = basePath
		}

		var templateWalker func(path string, info os.FileInfo, err error) error
		templateWalker = func(path string, info os.FileInfo, err error) error {
			if err != nil {
				ERROR.Println("error walking templates:", err)
				return nil
			}

			// is it a symlinked template?
			link, err := os.Lstat(path)
			if err == nil && link.Mode()&os.ModeSymlink == os.ModeSymlink {
				TRACE.Println("symlink template:", path)
				// lookup the actual target & check for goodness
				targetPath, err := filepath.EvalSymlinks(path)
				if err != nil {
					ERROR.Println("Failed to read symlink", err)
					return err
				}
				targetInfo, err := os.Stat(targetPath)
				if err != nil {
					ERROR.Println("Failed to stat symlink target", err)
					return err
				}

				// set the template path to the target of the symlink
				path = targetPath
				info = targetInfo

				// need to save state and restore for recursive call to Walk on symlink
				tmp := fullSrcDir
				fullSrcDir = filepath.Dir(targetPath)
				filepath.Walk(targetPath, templateWalker)
				fullSrcDir = tmp
			}

			// Walk into watchable directories
			if info.IsDir() {
				if !loader.WatchDir(info) {
					return filepath.SkipDir
				}
				return nil
			}

			// Only add watchable
			if !loader.WatchFile(info.Name()) {
				return nil
			}

			var fileStr string

			// addTemplate loads a template file into the Go template loader so it can be rendered later
			addTemplate := func(templateName string) (err error) {
				TRACE.Println("adding template: ", templateName)
				// Convert template names to use forward slashes, even on Windows.
				if os.PathSeparator == '\\' {
					templateName = strings.Replace(templateName, `\`, `/`, -1) // `
				}

				// If we already loaded a template of this name, skip it.
				lowerTemplateName := strings.ToLower(templateName)
				if _, ok := loader.templateNames[lowerTemplateName]; ok {
					return nil
				}

				loader.templatePaths[templateName] = path
				loader.templateNames[lowerTemplateName] = templateName

				// Load the file if we haven't already
				if fileStr == "" {
					fileBytes, err := ioutil.ReadFile(path)
					if err != nil {
						ERROR.Println("Failed reading file:", path)
						return nil
					}

					fileStr = string(fileBytes)
				}

				_, err = loader.templateSet.New(templateName).Parse(fileStr)
				return err
			}

			templateName := path[len(fullSrcDir)+1:]

			err = addTemplate(templateName)

			// Store / report the first error encountered.
			if err != nil && loader.compileError == nil {
				_, line, description := parseTemplateError(err)
				loader.compileError = &Error{
					Title:       "Template Compilation Error",
					Path:        templateName,
					Description: description,
					Line:        line,
					SourceLines: strings.Split(fileStr, "\n"),
				}
				ERROR.Printf("Template compilation error (In %s around line %d):\n%s",
					templateName, line, description)
			}
			return nil
		}

		funcErr := filepath.Walk(fullSrcDir, templateWalker)

		// If there was an error with the Funcs, set it and return immediately.
		if funcErr != nil {
			loader.compileError = funcErr.(*Error)
			if loader.compileError == nil {
				return nil
			} else {
				return loader.compileError
			}
		}
	}

	for _, i := range AssetNames() {
		lowerTemplateName := strings.ToLower(i)
		// If we already loaded a template of this name, skip it.
		if _, ok := loader.templateNames[lowerTemplateName]; ok {
			continue
		}

		if raw, err := Asset(i); err == nil {
			TRACE.Println("adding embedded template: ", i)
			if _, err := loader.templateSet.New(i).Parse(string(raw)); err != nil {
				ERROR.Printf("Error compiling embedded template %s: %s\n", i, err)
				continue
			}
			loader.templatePaths[i] = ""
			loader.templateNames[lowerTemplateName] = i
		}
	}

	if loader.compileError == nil {
		return nil
	} else {
		return loader.compileError
	}
}

func (loader *TemplateLoader) WatchDir(info os.FileInfo) bool {
	// Watch all directories, except the ones starting with a dot.
	return !strings.HasPrefix(info.Name(), ".")
}

func (loader *TemplateLoader) WatchFile(basename string) bool {
	// Watch all files, except the ones starting with a dot.
	return !strings.HasPrefix(basename, ".")
}

// Parse the line, and description from an error message like:
// html/template:Application/Register.html:36: no such template "footer.html"
func parseTemplateError(err error) (templateName string, line int, description string) {
	description = err.Error()
	i := regexp.MustCompile(`:\d+:`).FindStringIndex(description)
	if i != nil {
		line, err = strconv.Atoi(description[i[0]+1 : i[1]-1])
		if err != nil {
			ERROR.Println("Failed to parse line number from error message:", err)
		}
		templateName = description[:i[0]]
		if colon := strings.Index(templateName, ":"); colon != -1 {
			templateName = templateName[colon+1:]
		}
		templateName = strings.TrimSpace(templateName)
		description = description[i[1]+1:]
	}
	return templateName, line, description
}

// Return the Template with the given name.  The name is the template's path
// relative to a template loader root.
//
// An Error is returned if there was any problem with any of the templates.  (In
// this case, if a template is returned, it may still be usable.)
func (loader *TemplateLoader) Template(name string, funcMaps ...Args) (Template, error) {
	if loader == nil {
		return nil, errors.New("no template loader")
	}

	// Case-insensitive matching of template file name
	templateName := loader.templateNames[strings.ToLower(name)]

	var err error
	var tmpl *template.Template

	if loader.templateSet == nil {
		if err := loader.Refresh(); err != nil {
			return nil, fmt.Errorf("No template loader, unable to refresh: %s", err)
		}
	}

	// Look up and return the template.
	tmpl = loader.templateSet.Lookup(templateName)

	// This is necessary.
	// If a nil loader.compileError is returned directly, a caller testing against
	// nil will get the wrong result.  Something to do with casting *Error to error.
	if loader.compileError != nil {
		err = loader.compileError
	}

	if tmpl == nil && err == nil {
		WARN.Printf("Template %s not found.", name)
		return nil, fmt.Errorf("Template %s not found.", name)
	}

	var funcMap template.FuncMap
	for _, i := range funcMaps {
		if funcMap == nil {
			funcMap = template.FuncMap{}
		}
		for k, v := range i {
			funcMap[k] = v
		}
	}

	return goTemplateWrapper{tmpl, loader, funcMap}, err
}

// Reads the lines of the given file.
func readLines(filename string) ([]string, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(bytes), "\n"), nil
}

// Adapter for Go Templates.
type goTemplateWrapper struct {
	*template.Template
	loader  *TemplateLoader
	funcMap template.FuncMap
}

// return a 'mars.Template' from Go's template.
func (t goTemplateWrapper) Render(wr io.Writer, arg interface{}) error {
	if t.funcMap == nil {
		return t.Template.Execute(wr, arg)
	}

	return t.Template.Funcs(t.funcMap).Execute(wr, arg)
}

func (t goTemplateWrapper) Content() []string {
	content, _ := readLines(t.loader.templatePaths[t.Template.Name()])
	return content
}

var _ Template = goTemplateWrapper{}

/////////////////////
// Template functions
/////////////////////

// Return a url capable of invoking a given controller method:
// "Application.ShowApp 123" => "/app/123"
func ReverseUrl(args ...interface{}) (template.URL, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no arguments provided to reverse route")
	}

	action := args[0].(string)
	if action == "Root" {
		return template.URL(AppRoot), nil
	}
	actionSplit := strings.Split(action, ".")
	if len(actionSplit) != 2 {
		return "", fmt.Errorf("reversing '%s', expected 'Controller.Action'", action)
	}

	// Look up the types.
	var c Controller
	if err := c.SetAction(actionSplit[0], actionSplit[1]); err != nil {
		return "", fmt.Errorf("reversing %s: %s", action, err)
	}

	if len(c.MethodType.Args) < len(args)-1 {
		return "", fmt.Errorf("reversing %s: route defines %d args, but received %d",
			action, len(c.MethodType.Args), len(args)-1)
	}

	// Unbind the arguments.
	argsByName := make(map[string]string)
	for i, argValue := range args[1:] {
		Unbind(argsByName, c.MethodType.Args[i].Name, argValue)
	}

	return template.URL(MainRouter.Reverse(args[0].(string), argsByName).Url), nil
}

func Slug(text string) string {
	separator := "-"
	text = strings.ToLower(text)
	text = invalidSlugPattern.ReplaceAllString(text, "")
	text = whiteSpacePattern.ReplaceAllString(text, separator)
	text = strings.Trim(text, separator)
	return text
}
