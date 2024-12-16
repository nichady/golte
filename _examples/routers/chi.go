package main

import (
	"net/http"

	"examples/routers/build"

	"github.com/go-chi/chi/v5"
	"github.com/HazelnutParadise/sveltigo"
)

func chiRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(build.sveltigo) // register main sveltigo middleware

	r.Use(sveltigo.Layout("layout/main")) // use this layout for all routes
	r.Get("/", sveltigo.Page("page/home"))
	r.Get("/about", sveltigo.Page("page/about"))
	r.Get("/contact", sveltigo.Page("page/contact"))

	r.Route("/user", func(r chi.Router) {
		r.Use(sveltigo.Layout("layout/secondary")) // use this layout for only "/user/login" and "/user/profile"
		r.Get("/login", sveltigo.Page("page/login"))
		r.Get("/profile", func(w http.ResponseWriter, r *http.Request) { // pass props to the component
			sveltigo.RenderPage(w, r, "page/profile", map[string]any{
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
		sveltigo.RenderError(w, r, "Page not found", http.StatusNotFound)
	})

	return r
}
