# Mars

A lightweight web toolkit for the [Go language](http://www.golang.org).

[![GoDoc](http://godoc.org/github.com/roblillack/mars?status.svg)](https://pkg.go.dev/github.com/roblillack/mars)
[![Build Status](https://travis-ci.com/roblillack/mars.svg?branch=master)](https://travis-ci.com/github/roblillack/mars)
[![Windows build status](https://ci.appveyor.com/api/projects/status/og951w3majhmd13t/branch/master?svg=true)](https://ci.appveyor.com/project/roblillack/mars/branch/master)
[![Documentation Status](https://readthedocs.org/projects/mars/badge/?version=latest)](http://mars.readthedocs.org/en/latest/?badge=latest)
[![Coverage Status](https://coveralls.io/repos/github/roblillack/mars/badge.svg?branch=master)](https://coveralls.io/github/roblillack/mars?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/roblillack/mars)](https://goreportcard.com/report/github.com/roblillack/mars)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

- Latest Mars version: 1.0.4 (released August 4, 2020)
- Supported Go versions: 1.12 … 1.18

Mars is a fork of the fantastic, yet not-that-idiomatic-and-pretty-much-abandoned, [Revel framework](https://github.com/revel/revel). You might take a look at the corresponding documentation for the time being.

## Quick Start

Getting started with Mars is as easy as:

1. Adding the package to your project

   ```sh
   $ go get github.com/roblillack/mars
   ```

2. Creating an empty routes file in `conf/routes`

   ```sh
   $ mkdir conf; echo > conf/routes
   ```

3. Running the server as part of your main package

   ```go
   package main

   import "github.com/roblillack/mars"

   func main() {
     mars.Run()
   }
   ```

This essentially sets up an insecure server as part of your application that listens to HTTP (only) and responds to all requests with a 404. To learn where to go from here, please see the [Mars tutorial](http://mars.readthedocs.io/en/latest/getting-started/)

## Differences to Revel

The major changes since forking away from Revel are these:

- More idiomatic approach to integrating the framework into your application:
  - No need to use the `revel` command to build, run, package, or distribute your app.
  - Code generation (for registering controllers and reverse routes) is supported using the standard `go generate` way.
  - No runtime dependencies anymore. Apps using Mars are truly standalone and do not need access to the sources at runtime (default templates and mime config are embedded assets).
  - You are not forced into a fixed directory layout or package names anymore.
  - Removed most of the "path magic" that tried to determine where the sources of your application and revel are: No global `AppPath`, `ViewsPath`, `TemplatePaths`, `RevelPath`, and `SourcePath` variables anymore.
- Added support for Go 1.5+ vendoring.
- Vendor Mars' dependencies as Git submodules.
- Added support for [HTTP dual-stack mode](https://github.com/roblillack/mars/issues/6).
- Added support for [generating self-signed SSL certificates on-the-fly](https://github.com/roblillack/mars/issues/6).
- Added [graceful shutdown](https://godoc.org/github.com/roblillack/mars#OnAppShutdown) functionality.
- Added [CSRF protection](https://godoc.org/github.com/roblillack/mars#CSRFFilter).
- Integrated `Static` controller to support hosting plain HTML files and assets.
- Removed magic that automatically added template parameter names based on variable names in `Controller.Render()` calls using code generation and runtime introspection.
- Removed the cache library.
- Removed module support.
- Removed support for configurable template delimiters.
- Corrected case of render functions (`RenderXml` --> `RenderXML`).
- Fix generating reverse routes for some edge cases: Action parameter is called `args` or action parameter is of type `interface{}`.
- Fixed a [XSS vulnerability](https://github.com/roblillack/mars/issues/1).

## Documentation

- [Getting started with Mars](http://mars.readthedocs.io/en/latest/getting-started/)
- [Moving from Revel to Mars in 7 steps](http://mars.readthedocs.io/en/latest/migration/)

## Links

- [Code Coverage](http://gocover.io/github.com/roblillack/mars)
- [Go Report Card](http://goreportcard.com/report/roblillack/mars)
- [GoDoc](https://godoc.org/github.com/roblillack/mars)
