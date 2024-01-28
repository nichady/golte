package golte

import (
	"context"
	"io/fs"
	"net/http"
	"strings"

	"github.com/nichady/golte/render"
)

// New constructs a golte middleware from the given filesystem.
// The root of the filesystem should be the golte build directory.
//
// The returned middleware is used to add a render context to incoming requests.
// It will allow you to use Layout, AddLayout, Page, RenderPage, and Render.
// It should be mounted on the root of your router.
// The middleware should not be mounted on routes other than the root.
func New(fsys fs.FS) func(http.Handler) http.Handler {
	serverDir, err := fs.Sub(fsys, "server")
	if err != nil {
		panic(err)
	}

	clientDir, err := fs.Sub(fsys, "client")
	if err != nil {
		panic(err)
	}

	renderer := render.New(serverDir)
	assets := fileServer(clientDir)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/"+renderer.AppPath()+"/") {
				assets.ServeHTTP(w, r)
				return
			}

			scheme := "http"
			if r.TLS != nil {
				scheme += "s"
			}

			ctx := context.WithValue(r.Context(), contextKey{}, &RenderContext{
				Renderer: renderer,
				ErrPage:  "$$$GOLTE_DEFAULT_ERROR$$$",
				scdata: render.SvelteContextData{
					URL: scheme + "://" + r.Host + r.URL.String(),
				},
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
func RenderPage(w http.ResponseWriter, r *http.Request, component string, props map[string]any) {
	rctx := GetRenderContext(r)
	rctx.Components = append(rctx.Components, render.Entry{
		Comp:  component,
		Props: props,
	})
	rctx.Render(w)
}

// RenderError renders the current error page along with layouts.
// It will also write the status code to the header.
func RenderError(w http.ResponseWriter, r *http.Request, message string, status int) {
	rctx := GetRenderContext(r)
	entry := render.Entry{Comp: rctx.ErrPage, Props: map[string]any{
		"message": message,
		"status":  status,
	}}
	rctx.Components = append(rctx.Components, entry)
	rctx.Render(respWriterWrapper{w})
}

// respWriterWrapper is needed to prevent superfluous WriteHeader calls
type respWriterWrapper struct {
	http.ResponseWriter
}

func (w respWriterWrapper) WriteHeader(int) {
	w.ResponseWriter.WriteHeader(http.StatusInternalServerError)
}
