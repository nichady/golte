package main

import (
	"net/http"

	"github.com/HazelnutParadise/sveltigo"
	"github.com/gin-gonic/gin"
)

func ginRouter() http.Handler {
	// Gin doesn't have a function to wrap middleware, so define our own
	wrapMiddleware := func(middleware func(http.Handler) http.Handler) func(ctx *gin.Context) {
		return func(ctx *gin.Context) {
			middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx.Request = r
				ctx.Next()
			})).ServeHTTP(ctx.Writer, ctx.Request)
			if sveltigo.GetRenderContext(ctx.Request) == nil {
				ctx.Abort()
			}
		}
	}

	// since gin doesm't use stdlib-compatible signatures, we have to wrap them
	page := func(c string) gin.HandlerFunc {
		return gin.WrapH(sveltigo.Page(c))
	}
	layout := func(c string) gin.HandlerFunc {
		return wrapMiddleware(sveltigo.Layout(c))
	}

	r := gin.Default()

	r.Use(wrapMiddleware(build.sveltigo))
	r.Use(layout("layout/main"))

	r.GET("/", page("page/home"))
	r.GET("/about", page("page/about"))
	r.GET("/contact", page("page/contact"))

	g := r.Group("/user")
	g.Use(wrapMiddleware(sveltigo.Layout("layout/secondary")))
	g.GET("/login", page("page/login"))
	g.GET("/profile", func(ctx *gin.Context) {
		sveltigo.RenderPage(ctx.Writer, ctx.Request, "page/profile", map[string]any{
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
		sveltigo.RenderError(ctx.Writer, ctx.Request, "Page not found", http.StatusNotFound)
	})

	r.GET("/:placeholder", func(ctx *gin.Context) {
		sveltigo.RenderError(ctx.Writer, ctx.Request, "Page not found", http.StatusNotFound)
	})

	return r
}
