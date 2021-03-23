package mars

var WatchFilter = func(c *Controller, fc []Filter) {
	if MainWatcher != nil {
		err := MainWatcher.Notify()
		if err != nil {
			c.Result = c.RenderError(err)
			return
		}
	}
	fc[0](c, fc[1:])
}
