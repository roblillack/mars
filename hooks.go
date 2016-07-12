package mars

func runStartupHooks() {
	for _, hook := range startupHooks {
		hook()
	}
}

var startupHooks []func()

// Register a function to be run at app startup.
//
// The order you register the functions will be the order they are run.
// You can think of it as a FIFO queue.
// This process will happen after the config file is read
// and before the server is listening for connections.
//
// Ideally, your application should have only one call to init() in the file init.go.
// The reason being that the call order of multiple init() functions in
// the same package is undefined.
// Inside of init() call mars.OnAppStart() for each function you wish to register.
//
// Example:
//
//      // from: yourapp/app/controllers/somefile.go
//      func InitDB() {
//          // do DB connection stuff here
//      }
//
//      func FillCache() {
//          // fill a cache from DB
//          // this depends on InitDB having been run
//      }
//
//      // from: yourapp/app/init.go
//      func init() {
//          // set up filters...
//
//          // register startup functions
//          mars.OnAppStart(InitDB)
//          mars.OnAppStart(FillCache)
//      }
//
// This can be useful when you need to establish connections to databases or third-party services,
// setup app components, compile assets, or any thing you need to do between starting Mars and accepting connections.
//
func OnAppStart(f func()) {
	startupHooks = append(startupHooks, f)
}
