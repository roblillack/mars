package mars

import (
	"fmt"
	"runtime/debug"
)

// PanicFilter wraps the action invocation in a protective defer blanket that
// converts panics into 500 "Runtime Error" pages.
func PanicFilter(c *Controller, fc []Filter) {
	defer func() {
		if err := recover(); err != nil {
			e := &Error{
				Title:       "Runtime Error",
				Description: fmt.Sprint(err),
			}

			if DevMode {
				e.Stack = string(debug.Stack())
			}

			ERROR.Println(e, "\n", e.Stack)
			c.Result = c.RenderError(e)
		}
	}()
	fc[0](c, fc[1:])
}
