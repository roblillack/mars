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

        //go:generate mars-gen register-controllers ./controllers
        //go:generate mars-gen reverse-routes -n routes -o routes/routes.gen.go ./controllers

   Make sure to check in the generated sources, too. Run `mars-gen --help` for usage info.
6. Setup a main entry point for your server, for example like this:

       package main

       import (
           "flag"
           "path"
           "github.com/mycompany/myapp/controllers"
           "github.com/roblillack/mars"
       )

       func main() {
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

           mars.Run()
       }
7. Run `go generate && go build && ./myapp` and be happy.
