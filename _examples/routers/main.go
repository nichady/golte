package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/labstack/echo/v4"
	"github.com/nichady/golte"

	"examples/build"
)

func main() {
	// same app using different routers
	// comment/uncomment to try the different routers.

	r := chiRouter()
	// r := serveMuxRouter()
	// r := gorillaMuxRouter()
	// r := echoRouter()
	// r := ginRouter()

	fmt.Println("Serving on :8000")
	http.ListenAndServe(":8000", r)
}

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

func echoRouter() http.Handler {
	e := echo.New()

	e.Use(echo.WrapMiddleware(build.Golte))

	e.Use(echo.WrapMiddleware(golte.Layout("layout/main")))
	e.GET("/", echo.WrapHandler(golte.Page("page/home")))
	e.GET("/about", echo.WrapHandler(golte.Page("page/about")))
	e.GET("/contact", echo.WrapHandler(golte.Page("page/contact")))
	e.RouteNotFound("/*", func(c echo.Context) error {
		golte.RenderError(c.Response().Writer, c.Request(), "Page not found", http.StatusNotFound)
		return nil
	})

	g := e.Group("/user")
	g.Use(echo.WrapMiddleware(golte.Layout("layout/secondary")))
	g.GET("/login", echo.WrapHandler(golte.Page("page/login")))
	g.GET("/profile", func(c echo.Context) error {
		golte.RenderPage(c.Response().Writer, c.Request(), "page/profile", map[string]any{
			"username":   "john123",
			"realname":   "John Smith",
			"occupation": "Software Engineer",
			"age":        22,
			"email":      "johnsmith@example.com",
			"site":       "https://example.com",
			"searching":  true,
		})
		return nil
	})
	g.RouteNotFound("/*", func(c echo.Context) error {
		golte.RenderError(c.Response().Writer, c.Request(), "Page not found", http.StatusNotFound)
		return nil
	})

	return e
}

func ginRouter() http.Handler {
	wrapMiddleware := func(middleware func(http.Handler) http.Handler) func(ctx *gin.Context) {
		return func(ctx *gin.Context) {
			middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx.Request = r
				ctx.Next()
			})).ServeHTTP(ctx.Writer, ctx.Request)
			if golte.GetRenderContext(ctx.Request) == nil {
				ctx.Abort()
			}
		}
	}

	r := gin.Default()

	r.Use(wrapMiddleware(build.Golte))
	r.Use(wrapMiddleware(golte.Layout("layout/main")))

	r.GET("/", gin.WrapH(golte.Page("page/home")))
	r.GET("/about", gin.WrapH(golte.Page("page/about")))
	r.GET("/contact", gin.WrapH(golte.Page("page/contact")))

	g := r.Group("/user")
	g.Use(wrapMiddleware(golte.Layout("layout/secondary")))
	g.GET("/login", gin.WrapH(golte.Page("page/login")))
	g.GET("/profile", func(ctx *gin.Context) {
		golte.RenderPage(ctx.Writer, ctx.Request, "page/profile", map[string]any{
			"username":   "john123",
			"realname":   "John Smith",
			"occupation": "Software Engineer",
			"age":        22,
			"email":      "johnsmith@example.com",
			"site":       "https://example.com",
			"searching":  true,
		})
	})

	g.GET("/:placeholder", func(ctx *gin.Context) {
		golte.RenderError(ctx.Writer, ctx.Request, "Page not found", http.StatusNotFound)
	})

	r.GET("/:placeholder", func(ctx *gin.Context) {
		golte.RenderError(ctx.Writer, ctx.Request, "Page not found", http.StatusNotFound)
	})

	return r
}
