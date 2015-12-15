# Mars

A lightweight web toolkit for the [Go language](http://www.golang.org).

[![Build Status](https://secure.travis-ci.org/roblillack/mars.svg?branch=master)](http://travis-ci.org/roblillack/mars)

Mars is a fork of the fantastic, yet not-that-idiomatic-and-pretty-much-abandoned, [Revel framework](https://github.com/revel/revel). You might take a look at the corresponding documentation for the time being.

## Quick Start

Hah. Sorry, nothing here, yet.

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
- Removed the cache library.
- Removed module support.

## Links
- [Code Coverage](http://gocover.io/github.com/roblillack/mars)
- [Go Report Card](http://goreportcard.com/report/roblillack/mars)
