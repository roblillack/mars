package mars

var WatchFilter = func(c *Controller, fc []Filter) {
	if mainWatcher != nil {
		err := mainWatcher.Notify()
		if err != nil {
			c.Result = c.RenderError(err)
			return
		}
	}
	fc[0](c, fc[1:])
}
