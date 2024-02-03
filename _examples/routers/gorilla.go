package main

import (
	"net/http"

	"examples/routers/build"

	"github.com/gorilla/mux"
	"github.com/nichady/golte"
)

func gorillaMuxRouter() http.Handler {
	r := mux.NewRouter()

	r.Use(build.Golte)

	r.Use(golte.Layout("layout/main"))
	r.Handle("/", golte.Page("page/home"))
	r.Handle("/about", golte.Page("page/about"))
	r.Handle("/contact", golte.Page("page/contact"))

	s := r.PathPrefix("/user").Subrouter()
	s.Use(golte.Layout("layout/secondary"))
	s.Handle("/login", golte.Page("page/login"))
	s.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
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

	// gorilla/mux behaves a little differently than other routers.
	// Middlewares registered with mux.Use() do not run on unregistered routes.
	// The middleware needs to work on unregistered routes because the Golte middleware serves files.
	// It also causes layouts to not render when using mux.NotFoundHandler.
	// So we implement this hack: https://stackoverflow.com/a/56937571

	notFound := func(w http.ResponseWriter, r *http.Request) {
		golte.RenderError(w, r, "Page not found", http.StatusNotFound)
	}
	r.NotFoundHandler = r.NewRoute().HandlerFunc(notFound).GetHandler()
	s.NotFoundHandler = s.NewRoute().HandlerFunc(notFound).GetHandler()

	return r
}
