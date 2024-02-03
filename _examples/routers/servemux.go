package main

import (
	"net/http"

	"examples/routers/build"

	"github.com/nichady/golte"
)

func serveMuxRouter() http.Handler {
	r := http.NewServeMux()

	mainLayout := golte.Layout("layout/main")
	secondaryLayout := golte.Layout("layout/secondary")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			// *ServeMux treats "/" as a wildcard
			golte.RenderError(w, r, "Page not found", http.StatusNotFound)
			return
		}

		golte.RenderPage(w, r, "page/home", nil)
	})
	r.Handle("/about", golte.Page("page/about"))
	r.Handle("/contact", golte.Page("page/contact"))

	userRoutes := http.NewServeMux()
	userRoutes.HandleFunc("/user/", func(w http.ResponseWriter, r *http.Request) {
		golte.RenderError(w, r, "Page not found", http.StatusNotFound)
	})
	userRoutes.Handle("/user/login", golte.Page("page/login"))
	userRoutes.HandleFunc("/user/profile", func(w http.ResponseWriter, r *http.Request) {
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

	r.Handle("/user/", secondaryLayout(userRoutes))

	return build.Golte(mainLayout(r))
}
