package middleware

import pickle "github.com/telhawk-systems/telhawk-stack/findings/app/http"

func Auth(ctx *pickle.Context, next func() pickle.Response) pickle.Response {
	token := ctx.BearerToken()
	if token == "" {
		return ctx.Unauthorized("missing token")
	}

	// TODO: validate token and set auth info
	// ctx.SetAuth(claims)

	return next()
}
