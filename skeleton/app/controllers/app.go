package controllers

import "github.com/roblillack/mars"

type App struct {
	*mars.Controller
}

func (c App) Index() mars.Result {
	return c.Render()
}
