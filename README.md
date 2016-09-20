# Mars

A lightweight web toolkit for the [Go language](http://www.golang.org).

[![GoDoc](http://godoc.org/github.com/roblillack/mars?status.svg)](http://godoc.org/github.com/roblillack/mars)
[![Build Status](https://secure.travis-ci.org/roblillack/mars.svg?branch=master)](http://travis-ci.org/roblillack/mars)
[![Documentation Status](https://readthedocs.org/projects/mars/badge/?version=latest)](http://mars.readthedocs.org/en/latest/?badge=latest)
[![Coverage Status](https://coveralls.io/repos/github/roblillack/mars/badge.svg?branch=master)](https://coveralls.io/github/roblillack/mars?branch=master)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Mars is a fork of the fantastic, yet not-that-idiomatic-and-pretty-much-abandoned, [Revel framework](https://github.com/revel/revel). You might take a look at the corresponding documentation for the time being.

## Quick Start

Hah. Sorry, nothing here, yet. But if you want to switch a Revel project to Mars--see below.

## Differences to Revel

The major changes since forking away from Revel are these:

- More idiomatic approach to integrating the framework into your application:
    + No need to use the `revel` command to build, run, package, or distribute your app.
    + Code generation (for registering controllers and reverse routes) is supported using the standard `go generate` way.
    + No runtime dependencies anymore. Apps using Mars are truly standalone and do not need access to the sources at runtime (default templates and mime config are embedded assets).
    + You are not forced into a fixed directory layout or package names anymore.
    + Removed most of the "path magic" that tried to determine where the sources of your application and revel are: No global `AppPath`, `ViewsPath`, `TemplatePaths`, `RevelPath`, and `SourcePath` variables anymore.
- Added support for Go 1.5 vendor experiment.
- Vendor Mars' dependencies as Git submodules.
- Integrated `Static` controller to support hosting plain HTML files and assets.
- Removed magic that automatically added template parameter names based on variable names in `Controller.Render()` calls using code generation and runtime introspection.
- Removed the cache library.
- Removed module support.
- Removed support for configurable template delimiters.
- Corrected case of render functions (`RenderXml` --> `RenderXML`).
- Fix generating reverse routes for some edge cases: Action parameter is called `args` or action parameter is of type `interface{}`.
- Added [CSRF protection](https://godoc.org/github.com/roblillack/mars#CSRFFilter) functionality.
- Fixed a [XSS vulnerability](https://github.com/roblillack/mars/issues/1).

## Moving from Revel to Mars in 7 steps
1. Add the dependency:
   - Add `github.com/roblillack/mars` to your depedencies using the Go vendoring tool of your choice, or
   - Add said repository as a git submodule, or
   - Just run `go get github.com/roblillack/mars` (which is Go's “I'm feeling lucky” button)
2. Replace all occurences of the `revel` package with `mars`. This will mainly be import paths and
   action results (`mars.Result` instead of `revel.Result`), but also things like accessing the config
   or logging. You can pretty much automate this.
3. Fix the case for some of the rendering functions your code might call:
   - RenderJson -> RenderJSON
   - RenderJsonP -> RenderJSONP
   - RenderXml -> RenderXML
   - RenderHtml -> RenderHTML
4. Set a [Key](https://godoc.org/github.com/roblillack/mars#ValidationResult.Key) for all validation result,
   because Mars will _not_ guess this based on variable names. Something like `c.Validation.Required(email)` becomes
   `c.Validation.Required(email).Key("email")`
5. Install mars-gen using `go get github.com/roblillack/mars/cmd/mars-gen` and set it up for
   controller registration and reverse route generation by adding comments like these to one of Go files:
   ```
//go:generate mars-gen register-controllers ./controllers
//go:generate mars-gen reverse-routes -n routes -o routes/routes.gen.go ./controllers
   ```
   Make sure to check in the generated sources, too. Run `mars-gen --help` for usage info.
6. Setup a main entry point for your server, for example like this:
   ```
   package main

   import (
       "flag"
       "path"
       "github.com/mycompany/myapp/controllers"
       "github.com/roblillack/mars"
   )

   func main() {
       port := flag.Int("p", -1, "Port to listen on (default: use mars config)")
       mode := flag.String("m", "prod", "Runtime mode to select (default: prod)")
       flag.Parse()

       // This is the function `mars-gen register-controllers` generates:
       controllers.RegisterControllers()

       // Setup some paths to be compatible with the Revel way. Default is not to have an "app" directory below BasePath
       mars.ViewsPath = path.Join("app", "views")
       mars.ConfigFile = path.Join("app", "conf", "app.conf")
       mars.RoutesFile = path.Join("app", "conf", "routes")

       // Ok, we should never, ever, ever disable CSRF protection.
       // But to stay compatible with Revel's defaults ....
       // Read https://godoc.org/github.com/roblillack/mars#CSRFFilter about what to do to enable this again.
       mars.DisableCSRF = true

       // Reads the config, sets up template loader, creates router
       mars.InitDefaults(mode, ".")

       if *port == -1 {
           *port = mars.HttpPort
       }

       mars.Run(*port)
   }
   ```
7. Run `go generate && go build && ./myapp` and be happy.

## Links
- [Code Coverage](http://gocover.io/github.com/roblillack/mars)
- [Go Report Card](http://goreportcard.com/report/roblillack/mars)
- [GoDoc](https://godoc.org/github.com/roblillack/mars)
