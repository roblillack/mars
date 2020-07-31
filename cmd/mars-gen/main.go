package main

import (
	"fmt"
	"os"
	"path"
	"text/template"
	"time"

	"github.com/codegangsta/cli"

	"github.com/roblillack/mars"
)

func fatalf(layout string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, layout+"\n", args...)
	os.Exit(1)
}

func main() {
	app := cli.NewApp()
	app.HideVersion = true
	app.Name = "mars-gen"
	app.Usage = "Code generation tool for the Mars web framework"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: "Prints the names of the source files as they are parsed",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "register-controllers",
			Usage:  "Generates code to register your controllers with the framework",
			Action: registerControllers,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "n",
					Value: "RegisterControllers",
					Usage: "Function name to generate",
				},
				cli.StringFlag{
					Name:  "o",
					Value: "register_controllers.gen.go",
					Usage: "Name of the file to generate",
				},
			},
		},
		{
			Name:   "reverse-routes",
			Usage:  "Generates code that allows generating reverse routes",
			Action: reverseRoutes,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "n",
					Value: "routes",
					Usage: "Package name to generate",
				},
				cli.StringFlag{
					Name:  "o",
					Value: "routes.gen.go",
					Usage: "Path of the file to generate",
				},
			},
		},
	}

	app.Run(os.Args)
}

func registerControllers(ctx *cli.Context) {
	dir := "."
	if len(ctx.Args()) > 0 {
		dir = ctx.Args()[0]
	}

	sourceInfo, procErr := ProcessSource(dir, ctx.GlobalBool("v"))
	if procErr != nil {
		fatalf(procErr.Error())
	}

	generateSources(registerTemplate, path.Join(dir, ctx.String("o")), map[string]interface{}{
		"packageName":  sourceInfo.PackageName,
		"functionName": ctx.String("n"),
		"controllers":  sourceInfo.ControllerSpecs(),
		"ImportPaths":  sourceInfo.CalcImportAliases(),
		"time":         time.Now(),
	})
}

func reverseRoutes(ctx *cli.Context) {
	dir := "."
	if len(ctx.Args()) > 0 {
		dir = ctx.Args()[0]
	}

	sourceInfo, procErr := ProcessSource(dir, ctx.GlobalBool("v"))
	if procErr != nil {
		fatalf(procErr.Error())
	}

	generateSources(routesTemplate, ctx.String("o"), map[string]interface{}{
		"packageName": ctx.String("n"),
		"controllers": sourceInfo.ControllerSpecs(),
		"ImportPaths": sourceInfo.CalcImportAliases(),
		"time":        time.Now(),
	})
}

func generateSources(tpl, filename string, templateArgs map[string]interface{}) {
	sourceCode := mars.ExecuteTemplate(template.Must(template.New("").Parse(tpl)), templateArgs)

	if err := os.MkdirAll(path.Dir(filename), 0755); err != nil {
		fatalf("Unable to create dir: %v", err)
	}

	// Create the file
	file, err := os.Create(filename)
	if err != nil {
		fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	if _, err := file.WriteString(sourceCode); err != nil {
		fatalf("Failed to write to file: %v", err)
	}
}

const registerTemplate = `// DO NOT EDIT -- code generated by mars-gen
package {{.packageName}}

import (
	"reflect"
	"github.com/roblillack/mars"{{range $k, $v := $.ImportPaths}}
	{{$v}} "{{$k}}"{{end}}
)

var (
	// So compiler won't complain if the generated code doesn't reference reflect package...
	_ = reflect.Invalid
)

func {{.functionName}}() {
	{{range $i, $c := .controllers}}
	mars.RegisterController((*{{.StructName}})(nil),
		[]*mars.MethodType{
			{{range .MethodSpecs}}&mars.MethodType{
				Name: "{{.Name}}",
				Args: []*mars.MethodArg{ {{range .Args}}
					&mars.MethodArg{Name: "{{.Name}}", Type: reflect.TypeOf((*{{index $.ImportPaths .ImportPath | .TypeExpr.TypeName}})(nil)) },{{end}}
				},
			},
			{{end}}
		})
	{{end}}
}
`

const routesTemplate = `// DO NOT EDIT -- code generated by mars-gen
package {{.packageName}}

import (
	"github.com/roblillack/mars"{{range $k, $v := $.ImportPaths}}
	{{$v}} "{{$k}}"{{end}}
)

{{range $i, $c := .controllers}}
type t{{.StructName}} struct {}
var {{.StructName}} t{{.StructName}}

{{range .MethodSpecs}}
func (_ t{{$c.StructName}}) {{.Name}}({{range .Args}}
                {{.Name}}_ {{if .ImportPath}}{{index $.ImportPaths .ImportPath | .TypeExpr.TypeName}}{{else}}{{.TypeExpr.TypeName ""}}{{end}},{{end}}
                ) string {
        args := make(map[string]string)
        {{range .Args}}
        mars.Unbind(args, "{{.Name}}", {{.Name}}_){{end}}
        return mars.MainRouter.Reverse("{{$c.StructName}}.{{.Name}}", args).Url
}
{{end}}
{{end}}
`
