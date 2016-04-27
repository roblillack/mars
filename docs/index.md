# Mars: A lightweight web toolkit for the Go programming language

**WORK IN PROGRESS**

[Mars](https://github.com/roblillack/mars) is a fork of the fantastic, yet not-that-idiomatic-and-pretty-much-abandoned, [Revel framework](https://github.com/revel/revel). You might take a look at the corresponding documentation for the time being.

Mars provides the following functionality:

…

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
- Corrected case of render functions (`RenderXml` --> `RenderXML`).
- Fix generating reverse routes for some edge cases: Action parameter is called `args` or action parameter is of type `interface{}`.