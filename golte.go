package golte

import (
	"context"
	"io"
	"io/fs"
	"net/http"

	"github.com/nichady/golte/render"
)

type Options struct {
	// HandleRenderError is the function called whenever there is an error in rendering.
	// This will be called before rendering error pages.
	// It is recommended to put things like logging here.
	HandleRenderError func(*http.Request, []render.Entry, error)
}

// From takes a filesystem and returns two things: a middleware and an http handler.
// The given filesystem should contain the build files of "npx golte".
// If not, this functions panics.
//
// The returned middleware is used to add a render context to incoming requests.
// It will allow you to use Layout, AddLayout, Page, RenderPage, and Render.
// It should be mounted on the route which you plan to serve your app (typically the root).

// The http handler is a file server that will serve JS, CSS, and other assets.
// It should be served on the same path as what you set "appPath" to in golte.config.js.
func From(fsys fs.FS, opts Options) (middleware func(http.Handler) http.Handler, assets http.HandlerFunc) {
	if opts.HandleRenderError == nil {
		opts.HandleRenderError = func(*http.Request, []render.Entry, error) {}
	}

	serverDir, err := fs.Sub(fsys, "server")
	if err != nil {
		panic(err)
	}

	clientDir, err := fs.Sub(fsys, "client")
	if err != nil {
		panic(err)
	}

	renderer := render.New(serverDir)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), contextKey{}, &RenderContext{
				Renderer:          renderer,
				HandleRenderError: opts.HandleRenderError,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, http.StripPrefix("/", fileServer(clientDir)).ServeHTTP
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

// Page returns a handler that will call RenderPage.
// Use this when there are no props needed to render the component.
// If complex logic and props are needed, instead use RenderPage.
func Page(component string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RenderPage(w, r, component, nil)
	})
}

// Error is a middleware which calls SetError.
// Use this when there are no props needed to render the component.
// If complex logic and props are needed, instead use SetError.
func Error(component string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			SetError(r, component)
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

	rctx := GetRenderContext(r)
	rctx.Layouts = append(rctx.Layouts, render.Entry{
		Comp:  component,
		Props: props,
	})
}

// SetError sets the error page for the route.
func SetError(r *http.Request, component string) {
	rctx := GetRenderContext(r)
	rctx.ErrPage = component
}

// RenderPage renders the specified component with props to the writer, along with
// any other components added to the request's context.
// The props must consist only of values that can be serialized as JSON.
// If an error occurs in rendering, it will render the current error page.
func RenderPage(w http.ResponseWriter, r *http.Request, component string, props map[string]any) {
	rctx := GetRenderContext(r)
	page := render.Entry{Comp: component, Props: props}
	entries := append(rctx.Layouts, page)

	err := rctx.Renderer.Render(w, entries, r.Header["Golte"] != nil)
	if err != nil {
		rctx.HandleRenderError(r, entries, err)
		if rerr, ok := err.(*render.RenderError); ok {
			rctx.Layouts = rctx.Layouts[:rerr.Index]
			RenderErrorPage(w, r, err.Error(), http.StatusInternalServerError)
		} else {
			// this shouldn't happen
			renderFallback(w, r, err.Error(), http.StatusInternalServerError)
		}
	}
}

// Render renders all the components in the request's context to the writer.
// If an error occurs in rendering, it will render the current error page.
func Render(w http.ResponseWriter, r *http.Request) {
	rctx := GetRenderContext(r)

	err := rctx.Renderer.Render(w, rctx.Layouts, r.Header["Golte"] != nil)
	if err != nil {
		rctx.HandleRenderError(r, rctx.Layouts, err)
		if rerr, ok := err.(*render.RenderError); ok {
			rctx.Layouts = rctx.Layouts[:rerr.Index]
			RenderErrorPage(w, r, err.Error(), http.StatusInternalServerError)
		} else {
			// this shouldn't happen
			renderFallback(w, r, err.Error(), http.StatusInternalServerError)
		}
	}
}

// RenderErrorPage renders the current error page.
// If an error occurs while rendering the error page, the fallback error page is used instead.
func RenderErrorPage(w http.ResponseWriter, r *http.Request, message string, status int) {
	w.WriteHeader(status)

	rctx := GetRenderContext(r)
	page := render.Entry{Comp: rctx.ErrPage, Props: map[string]any{
		"message": message,
		"code":    status,
	}}
	entries := append(rctx.Layouts, page)

	err := rctx.Renderer.Render(w, entries, r.Header["Golte"] != nil)
	if err != nil {
		rctx.HandleRenderError(r, entries, err)
		renderFallback(w, r, err.Error(), -1)
	}
}

// renderFallback renders the fallback error page, an html template.
func renderFallback(w http.ResponseWriter, r *http.Request, message string, status int) {
	// rctx := getRenderContext(r)
	// TODO

	if status != -1 {
		w.WriteHeader(status)
	}
	io.WriteString(w, message)
}
