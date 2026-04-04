package controllers

import (
	pickle "github.com/telhawk-systems/telhawk-stack/findings/app/http"
)

type WelcomeController struct {
	pickle.Controller
}

func (c WelcomeController) Index(ctx *pickle.Context) pickle.Response {
	return ctx.JSON(200, map[string]string{
		"message": "Welcome to Pickle!",
	})
}
