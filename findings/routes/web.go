package routes

import (
	pickle "github.com/telhawk-systems/telhawk-stack/findings/app/http"
	"github.com/telhawk-systems/telhawk-stack/findings/app/http/controllers"
	"github.com/telhawk-systems/telhawk-stack/findings/app/http/middleware"
)

var API = pickle.Routes(func(r *pickle.Router) {
	r.Group("/api/v1", func(r *pickle.Router) {
		r.Get("/scans", controllers.ScanController{}.Index)
		r.Post("/scans", controllers.ScanController{}.Store)
		r.Get("/scans/:id", controllers.ScanController{}.Show)
		r.Delete("/scans/:id", controllers.ScanController{}.Destroy)

		r.Get("/findings", controllers.FindingController{}.Index)
		r.Get("/findings/:id", controllers.FindingController{}.Show)
	}, middleware.Auth)
})
