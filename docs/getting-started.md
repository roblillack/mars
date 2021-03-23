# Getting started with Mars

**WORK IN PROGRESS**

There is _no_ fixed directory hierarchy with Mars, but a projects' structure typically looks like this:

```
- myProject (Your project)
  |
  |-- main.go (Your main go code might live here, but can also be in ./cmd/something)
  |
  |-- conf (Directory with configuration files which are needed at runtime)
  |    |
  |    |-- app.conf (main configuration file)
  |    |
  |    +-- routes (Configuration of the routes)
  |
  |-- views (All the view templates are here)
  |    |
  |    |-- hotel (view templates for the “Hotel” controller)
  |    |
  |    +-- other (view templates for the “Other” controller)
  |
  +-- mySubpackage (Your code is in arbitrary sub packages)
       |
       +-- foo.go
```
