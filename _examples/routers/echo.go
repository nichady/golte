package main

import (
	"net/http"

	"examples/routers/build"

	"github.com/labstack/echo/v4"
	"github.com/nichady/golte"
)

func echoRouter() http.Handler {
	// since echo doesm't use stdlib-compatible signatures, we have to wrap them
	layout := func(c string) echo.MiddlewareFunc {
		return echo.WrapMiddleware(golte.Layout(c))
	}
	page := func(c string) echo.HandlerFunc {
		return echo.WrapHandler(golte.Page(c))
	}

	e := echo.New()

	e.Use(echo.WrapMiddleware(build.Golte))

	e.Use(layout("layout/main"))
	e.GET("/", page("page/home"))
	e.GET("/about", page("page/about"))
	e.GET("/contact", page("page/contact"))
	e.RouteNotFound("/*", func(c echo.Context) error {
		golte.RenderError(c.Response().Writer, c.Request(), "Page not found", http.StatusNotFound)
		return nil
	})

	g := e.Group("/user")
	g.Use(layout("layout/secondary"))
	g.GET("/login", page("page/login"))
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
