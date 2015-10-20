package app

import "github.com/roblillack/mars"

func init() {
	// Filters is the default set of global filters.
	mars.Filters = []mars.Filter{
		mars.PanicFilter,             // Recover from panics and display an error page instead.
		mars.RouterFilter,            // Use the routing table to select the right Action
		mars.FilterConfiguringFilter, // A hook for adding or removing per-Action filters.
		mars.ParamsFilter,            // Parse parameters into Controller.Params.
		mars.SessionFilter,           // Restore and write the session cookie.
		mars.FlashFilter,             // Restore and write the flash cookie.
		mars.ValidationFilter,        // Restore kept validation errors and save new ones from cookie.
		mars.I18nFilter,              // Resolve the requested language
		HeaderFilter,                 // Add some security based headers
		mars.InterceptorFilter,       // Run interceptors around the action.
		mars.CompressFilter,          // Compress the result.
		mars.ActionInvoker,           // Invoke the action.
	}

	// register startup functions with OnAppStart
	// ( order dependent )
	// mars.OnAppStart(InitDB)
	// mars.OnAppStart(FillCache)
}

// TODO turn this into mars.HeaderFilter
// should probably also have a filter for CSRF
// not sure if it can go in the same filter or not
var HeaderFilter = func(c *mars.Controller, fc []mars.Filter) {
	// Add some common security headers
	c.Response.Out.Header().Add("X-Frame-Options", "SAMEORIGIN")
	c.Response.Out.Header().Add("X-XSS-Protection", "1; mode=block")
	c.Response.Out.Header().Add("X-Content-Type-Options", "nosniff")

	fc[0](c, fc[1:]) // Execute the next filter stage.
}
