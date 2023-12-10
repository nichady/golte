package golte

import (
	"context"
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
			scheme := "http"
			if r.TLS != nil {
				scheme += "s"
			}

			ctx := context.WithValue(r.Context(), contextKey{}, &RenderContext{
				Renderer:          renderer,
				HandleRenderError: opts.HandleRenderError,
				scdata: render.SvelteContextData{
					URL: scheme + "://" + r.Host + r.URL.String(),
				},
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, http.StripPrefix("/", fileServer(clientDir)).ServeHTTP
}

// Layout returns a middleware that calls AddLayout.
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

// Error returns a middleware that calls SetError.
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

// Page returns a handler that calls RenderPage.
// Use this when there are no props needed to render the component.
// If complex logic and props are needed, instead use RenderPage.
func Page(component string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RenderPage(w, r, component, nil)
	})
}

// AddLayout appends the component to the request.
// The props must consist only of values that can be serialized as JSON.
func AddLayout(r *http.Request, component string, props map[string]any) {
	rctx := GetRenderContext(r)
	rctx.Components = append(rctx.Components, render.Entry{
		Comp:  component,
		Props: props,
	})
}

// SetError sets the error page for the request.
func SetError(r *http.Request, component string) {
	GetRenderContext(r).ErrPage = component
}

// RenderPage renders the specified component along with any layouts.
// The props must consist only of values that can be serialized as JSON.
// If an error occurs in rendering, it will render the current error page.
func RenderPage(w http.ResponseWriter, r *http.Request, component string, props map[string]any) {
	rctx := GetRenderContext(r)
	rctx.Components = append(rctx.Components, render.Entry{
		Comp:  component,
		Props: props,
	})
	rctx.Render(w)
}

// RenderErrorPage renders the current error page along with layouts..
// It will also write the status code to the header.
func RenderErrorPage(w http.ResponseWriter, r *http.Request, message string, status int) {
	GetRenderContext(r).RenderErrorPage(w, message, status)
}
