package main

import (
	"net/http"

	"examples/routers/build"

	"github.com/go-chi/chi/v5"
	"github.com/nichady/golte"
)

func chiRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(build.Golte) // register main Golte middleware

	r.Use(golte.Layout("layout/main")) // use this layout for all routes
	r.Get("/", golte.Page("page/home"))
	r.Get("/about", golte.Page("page/about"))
	r.Get("/contact", golte.Page("page/contact"))

	r.Route("/user", func(r chi.Router) {
		r.Use(golte.Layout("layout/secondary")) // use this layout for only "/user/login" and "/user/profile"
		r.Get("/login", golte.Page("page/login"))
		r.Get("/profile", func(w http.ResponseWriter, r *http.Request) { // pass props to the component
			golte.RenderPage(w, r, "page/profile", map[string]any{
				"username":   "john123",
				"realname":   "John Smith",
				"occupation": "Software Engineer",
				"age":        22,
				"email":      "johnsmith@example.com",
				"site":       "https://example.com",
				"searching":  true,
			})
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		golte.RenderError(w, r, "Page not found", http.StatusNotFound)
	})

	return r
}
