# MARS CHANGELOG

All notable changes to Mars will be documented in this file.
The format is based on [Keep a Changelog](http://keepachangelog.com/).

## [Unreleased](https://github.com/roblillack/mars/compare/v1.0.4...master)

- Add unit test for `cert`.
- Improve code generation to not write redundant type information. #20

## [v1.0.4](https://github.com/roblillack/mars/compare/v1.0.3...v1.0.4)

- Code generation:
  - Add package name cache to speed up parsing very large code bases. #19
  - Add `--verbose` or `-v` flag to print the names of source files as they are parsed. #19

## [v1.0.3](https://github.com/roblillack/mars/compare/v1.0.2...v1.0.3)

- Infrastructure:
  - Remove support for Go 1.8 to 1.11, depend on Go Modules by default.
- Usage:
  - Introduce automatic `setup` process for TemplateLoader and Router. Mars can now be started without calling [InitDefaults()](https://godoc.org/github.com/roblillack/mars#InitDefaults) before. #13
- Code generation:
  - Add functional tests. #19
  - Improve parsing speed, by skipping unnecessary imports of the Go standard library. #19

## [v1.0.2](https://github.com/roblillack/mars/compare/v1.0.1...v1.0.2)

- Defaults:
  - Fix mime type for Python bytecode.
- Infrastructure:
  - Add AppVeyor for automated builds & tests on Windows.
  - Fix tests when running in Go Modules mode.
  - Fix `got vet ./...` issues.

## [v1.0.1](https://github.com/roblillack/mars/compare/v1.0.0...v1.0.1)

- Router: Fix panic, if no router initialized. #10
- Templates:
  - Fix panic, if no template loader initialized. #11
  - Setup fallback template loader to use embedded templates without configuration. #12

## [v1.0.0](https://github.com/roblillack/mars/compare/a9a2ff4...v1.0.0)

- Let's make that 1.0.0.
- Build with current Go versions.
- Setup Go Module.
- Remove old versioning information.
- Remove git submodules.
- mars-gen: More compiler error fixes.
- Fix Go tip compiler errors.
- Router: Add support for file extensions after actions argutments. #9
- Enable graceful shutdown. #8
- Implement shutdown hooks. #8
- Remove support for Go <1.8. #8
- README: Fix links
- router: Fix path escaping for Go<1.8. #7
- travis: Add Go 1.8 support.
- router: Fix building reverse routes with path segments that contain reserved characters. #7
- compress: Compress SVG images, too.
- README: Document changes form #6.
- Add HTTP(S) dualstack support (incl. option to generate self-signed certs). #6
- mars-gen: Sort files by name when processing packages to get stable order of registered controllers.
- mars-gen: Make sure, we can generate without a resolvable Mars installation. Fixes #5.
- Merge pull request #4 from ipoerner/issue-3
- Default to no timeout with configurable changes.
- Allow setting absolute ConfigFile path.
- Fix panic_test for Go 1.5.
- Add Go 1.7 to travis builds.
- panic_test: Make go 1.5 error easier to debug.
- Add test for panic filter.
- templates: Add template availability test.
- Fix context-aware translation function, add test. #2
- templates: Fix HTML safe render func, add tests. #2
- templates: First work towards HTML-safe translate func. #2
- Document XSS fix #1.
- templates: Stop implicitly marking translation output as safe. Fixes #1.
- docs: Add testing.md
- CSRF protection: Set debugging messages to trace, fix 'SkipCSRF' check.
- README: Update to reflect CSRF changes.
- csrf: Make this an actual func.
- docs: Update to reflect CSRF stuff.
- Add CSRF protection functionality.
- server: Expose mars.Handler, also fixes refreshing Router.
- server: Allow booting without having a config file.
- Get rid of configurable template delimiters.
- mars-gen: Fix parsing array types.
- Add Coverage Status badge to README
- Add coveralls integration
- server: Don't set up watcher for templates, if we have no TemplateLoader.
- travis: Build with Go 1.5, 1.6, and tip.
- Remove "cron" dependency – not used anymore.
- Remove glide from .gitmodules.
- Change submodule repo paths to match glide config's.
- Update fsnotify and x/net submodules.
- Update fsnotify to v1.3.1, remove gopkg.in dependency.
- Reformat glide.yaml.
- docs: Fix formatting.
- docs: Start working on the documentation.
- Streamline logger configuration.
- Fix fakeapp test.
- Better handling of default values.
- Remove unused code.
- Remove Initialized flag.
- Remove CodePaths, ConfPaths, ImportPath.
- README: Document how to switch from Revel to Mars.
- Sort controllers, when generating code to not pollute your 'git status' all the time.
- Fix overriding embedded templates from application provided ones.
- Add caching functionality to Static controller.
- README: Add GoDoc reference image.
- README: Update regarding the reverse route generation fixes.
- mars-gen: Remove debug messages.
- mars-gen: Add support for action parameters of type interface{}
- mars-gen: Add support for action parameters called "args".
- Further improve documentation of interception functionality.
- Add render function changes to README.
- Add documentation to interception handling.
- Fix case for render functions and result types.
- Code style improvements.
- Remove magic, that adds template parameters based on variable names in Render calls.
- README: Add GoDoc reference.
- Start improving code style to make golint happier.
- README: Document differences to revel.
- Add mars-gen -- the code generator for registering controllers and reverse routes.
- Remove revel modules.
- Fix fakeapp test.
- Add some links to README.
- Add static controller (was a module before).
- templates: Fix preloading of embedded templates.
- Remove module support.
- Remove AppPath, ViewsPath, TemplatePaths.
- template: Remove possibility of specifying delimiters.
- fix build.
- travis: Fix vendor experiment.
- Add .gitmodules.
- travis: Remove glide, dependencies are Git submodules.
- travis: Enable Go Vendor Experiment
- Enable travis.
- More renaming.
- Remove skeleton. Should really be another repository.
- init: Start removing path magic, RevelPath + SourcePath is gone.
- mime: Embed default mime-types.
- Fix tests.
- template: Add support for embedded error templates.
- router: Remove automatically reading routes file.
- controller: Set HTTP Status OK only after successfully loading template.
- Remove cache/.
- Vendor dependencies using Glide.
- Add support for Go 1.5 Vendor Experiment.
- Rename package to 'github.com/roblillack/mars'.
