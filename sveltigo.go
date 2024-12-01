package sveltigo

import (
	"context"
	"io/fs"
	"net/http"
	"strings"

	"github.com/HazelnutParadise/sveltigo/render"
)

var assetsDir *fs.FS

func SetAssetsDir(dir *fs.FS) {
	assetsDir = dir
}

var Mode string

const (
	RenderModeCSR = "CSR"
	RenderModeSSR = "SSR"
)

func SetMode(mode string) {
	Mode = mode
}

// Props is an alias for map[string]any. It exists for documentation purposes.
// Props must be JSON-serializable when passing to fuctions defined in this package.
type Props = map[string]any

// New constructs a golte middleware from the given filesystem.
// The root of the filesystem should be the golte build directory.
//
// The returned middleware is used to add a render context to incoming requests.
// It will allow you to use [Layout], [AddLayout], [Page], and [RenderPage].
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

	renderer := render.New(&serverDir, &clientDir, Mode)
	assets := http.StripPrefix("/"+renderer.Assets()+"/", fileServer(clientDir))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/"+renderer.Assets()+"/") {
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

// Layout returns a middleware that calls [AddLayout].
// Use this when there are no props needed to render the component.
// If you need to pass props, use [AddLayout] instead.
func Layout(component string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			AddLayout(r, component, nil)
			next.ServeHTTP(w, r)
		})
	}
}

// Error returns a middleware that calls [SetError].
// Use this when there are no props needed to render the component.
// If you need to pass props, use [SetError] instead.
func Error(component string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			SetError(r, component)
			next.ServeHTTP(w, r)
		})
	}
}

// Page returns a handler that calls [RenderPage].
// Use this when there are no props needed to render the component.
// If you need to pass props, use [RenderPage] instead.
func Page(component string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RenderPage(w, r, component, nil)
	})
}

// AddLayout appends the component to the request.
// Layouts consist of any components with a <slot>.
// Calling this multiple times on the same request will nest layouts.
func AddLayout(r *http.Request, component string, props Props) {
	rctx := MustGetRenderContext(r)
	rctx.Components = append(rctx.Components, render.Entry{
		Comp:  component,
		Props: props,
	})
}

// SetError sets the error page for the request.
// Errors consist of any components that take the "message" and "status" props.
// Calling this multiple times on the same request will overrite the previous error page.
func SetError(r *http.Request, component string) {
	MustGetRenderContext(r).ErrPage = component
}

// RenderPage renders the specified component.
// If any layouts were added previously, then each subsequent layout will
// go in the <slot> of the previous layout. The page will be in the <slot>
// of the last layout.
func RenderPage(w http.ResponseWriter, r *http.Request, component string, props Props) {
	rctx := MustGetRenderContext(r)
	rctx.Components = append(rctx.Components, render.Entry{
		Comp:  component,
		Props: props,
	})
	rctx.Render(w)
}

// RenderError renders the current error page along with layouts.
// The error componenet will receive "message" and "status" as props.
// It will also write the status code to the header.
func RenderError(w http.ResponseWriter, r *http.Request, message string, status int) {
	rctx := MustGetRenderContext(r)
	entry := render.Entry{Comp: rctx.ErrPage, Props: Props{
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
