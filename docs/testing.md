# Testing with Mars

As Mars tries to achieve a more idiomatic approach to devloping web applications with Go as Revel does,
unit tests are written using the standard Go `testing` package.

On top of this, Mars provides an easy to use TestSuite (github.com/mars/testing)[https://godoc.org/github.com/roblillack/mars/testing]
which can be used like this:

    package controllers

    import (
        "os"
        "path/filepath"
        "runtime"
        "testing"
        "time"

        "github.com/roblillack/mars"
        marst "github.com/roblillack/mars/testing"
    )

    func TestMain(m *testing.M) {
        setupMars()
        retCode := m.Run()
        os.Exit(retCode)
    }

    func setupMars() {
        _, filename, _, _ := runtime.Caller(0)

        RegisterControllers()
        mars.ViewsPath = filepath.Join("app", "views")
        mars.InitDefaults("dev", filepath.Join(filepath.Dir(filename), "..", ".."))
        mars.DevMode = true

        go mars.Run()

        time.Sleep(1 * time.Second)
    }

    func Test_Health(t *testing.T) {
        ts := marst.NewTestSuite()
        ts.Get("/health")
        ts.AssertContains("Ok")
        ts.AssertOk()
    }
