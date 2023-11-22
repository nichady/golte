package golte

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/nichady/golte/render"
)

type Options struct {
	// AssetsPath is the absolute path from which asset files will be served.
	AssetsPath string

	// RenderErrorHandler is the function called whenever there is an error in rendering.
	// This will be called before rendering error pages.
	// It is recommended to put things like logging here.
	// index is the index of the entry in entries which caused the error
	RenderErrorHandler func(url string, entries []render.Entry, index int, err error)
}

// From takes a filesystem and returns two things: a middleware and an http handler.
// The given filesystem should contain the build files of "npx golte".
// If not, this functions panics.
//
// The returned middleware is used to add a render context to incoming requests.
// It will allow you to use Layout, AddLayout, Page, RenderPage, and Render.
// It should be mounted on the route which you plan to serve your app (typically the root).

// The http handler is a file server that will serve assets, such as js and css files.
// It should typically be served on a subpath of your app rather than the root.
// If you do choose to serve it on a subpath, make sure to set Options.AssetsPath as well.
func From(fsys fs.FS, opts Options) (middleware func(http.Handler) http.Handler, assets http.HandlerFunc) {
	if !strings.HasPrefix(opts.AssetsPath, "/") {
		opts.AssetsPath = "/" + opts.AssetsPath
	}

	if !strings.HasSuffix(opts.AssetsPath, "/") {
		opts.AssetsPath = opts.AssetsPath + "/"
	}

	if opts.RenderErrorHandler == nil {
		opts.RenderErrorHandler = func(string, []render.Entry, int, error) {}
	}

	serverDir, err := fs.Sub(fsys, "server")
	if err != nil {
		panic(err)
	}

	clientDir, err := fs.Sub(fsys, "client")
	if err != nil {
		panic(err)
	}

	renderer := render.New(serverDir, opts.AssetsPath)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), contextKey{}, &renderContext{
				renderer:           renderer,
				renderErrorHandler: opts.RenderErrorHandler,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, http.StripPrefix(opts.AssetsPath, fileServer(clientDir)).ServeHTTP
}

// Layout returns a middleware that will add the specified component to the context.
// Use this when there are no props needed to render the component.
// If complex logic and props are needed, instead use AddLayout.
func Layout(component string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			AddLayout(r, component, nil)
			next.ServeHTTP(w, r)
		})
	}
}

// Page returns a handler that will render the specified component, along with
// any other components added to the request's context.
// Use this when there are no props needed to render the component.
// If complex logic and props are needed, instead use RenderPage.
func Page(component string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RenderPage(w, r, component, nil)
	})
}

// Error is a middleware which sets the error page for the route.
func Error(component string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rctx := getRenderContext(r)
			rctx.errpage = component
			next.ServeHTTP(w, r)
		})
	}
}

// AddLayout adds the specified component with props to the request's context.
// The props must consist only of values that can be serialized as JSON.
func AddLayout(r *http.Request, component string, props map[string]any) {
	if props == nil {
		props = map[string]any{}
	}

	rctx := getRenderContext(r)
	rctx.layouts = append(rctx.layouts, render.Entry{
		Comp:  component,
		Props: props,
	})
}

// RenderPage renders the specified component with props to the writer, along with
// any other components added to the request's context.
// The props must consist only of values that can be serialized as JSON.
// If an error occurs in rendering, it will render the current error page.
func RenderPage(w http.ResponseWriter, r *http.Request, component string, props map[string]any) {
	rctx := getRenderContext(r)
	page := render.Entry{Comp: component, Props: props}
	entries := append(rctx.layouts, page)

	err := rctx.renderer.Render(w, entries)
	if err != nil {
		rerr, ok := rctx.renderer.ToRenderError(err)
		if ok {
			rctx.renderErrorHandler(r.URL.String(), entries, rerr.Index, err)
			rctx.layouts = rctx.layouts[:rerr.Index]
			RenderErrorPage(w, r, rerr.Cause.String(), http.StatusInternalServerError)
		} else {
			// this shouldn't happen
			rctx.renderErrorHandler(r.URL.String(), entries, -1, err)
			RenderFallbackPage(w, r, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Render renders all the components in the request's context to the writer.
// If an error occurs in rendering, it will render the current error page.
func Render(w http.ResponseWriter, r *http.Request) {
	rctx := getRenderContext(r)

	err := rctx.renderer.Render(w, rctx.layouts)
	if err != nil {
		rerr, ok := rctx.renderer.ToRenderError(err)
		if ok {
			rctx.renderErrorHandler(r.URL.String(), rctx.layouts, rerr.Index, err)
			rctx.layouts = rctx.layouts[:rerr.Index]
			RenderErrorPage(w, r, rerr.Cause.String(), http.StatusInternalServerError)
		} else {
			// this shouldn't happen
			rctx.renderErrorHandler(r.URL.String(), rctx.layouts, -1, err)
			RenderFallbackPage(w, r, err.Error(), http.StatusInternalServerError)
		}
	}
}

// RenderErrorPage renders the current error page.
// If an error occurs while rendering the error page, the fallback error page is used instead.
func RenderErrorPage(w http.ResponseWriter, r *http.Request, message string, status int) {
	rctx := getRenderContext(r)
	page := render.Entry{Comp: rctx.errpage, Props: map[string]any{
		"message": message,
		"code":    status,
	}}
	entries := append(rctx.layouts, page)

	err := rctx.renderer.Render(w, entries)
	if err != nil {
		rerr, ok := rctx.renderer.ToRenderError(err)
		if !ok {
			rctx.renderErrorHandler(r.URL.String(), entries, rerr.Index, err)
		} else {
			rctx.renderErrorHandler(r.URL.String(), entries, -1, err)
		}
		RenderFallbackPage(w, r, err.Error(), http.StatusInternalServerError)
	}
}

// RenderFallbackPage renders the fallback error page, an html template.
func RenderFallbackPage(w http.ResponseWriter, r *http.Request, message string, status int) {
	// rctx := getRenderContext(r)
	// TODO

	w.WriteHeader(status)
	io.WriteString(w, message)
}
