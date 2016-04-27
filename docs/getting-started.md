# Getting started with Mars

**WORK IN PROGRESS**

There is _no_ fixed directory hierarchy with Mars, but a projects' structure typically looks like this:

```
- $GOPATH/src/myProject (Your project)
  |
  |-- main.go (Your main go code can live here)
  |
  |-- conf (Directory with configuration files which are needed at runtime)
  |    |
  |    |-- app.conf (main configuration file)
  |    |
  |    |-- routes (Configuration of the routes)
  |
  |-- views (All the view templates are here)
  |    |
  |    |-- hotel (main configuration file)
  |    |
  |    |-- routes (Configuration of the routes)
  |
  |-- mySubpackage (Your code is in arbitrary sub packages)
  |    |
  |    |-- foo.go
  |
  |-- vendor (Your vendored dependencies are here)
       |-- ...
```