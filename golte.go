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
	AssetsPath string
}

func New(fsys fs.FS, opts Options) (middleware func(http.Handler) http.Handler, assets http.HandlerFunc) {
	if !strings.HasPrefix(opts.AssetsPath, "/") {
		opts.AssetsPath = "/" + opts.AssetsPath
	}

	if !strings.HasSuffix(opts.AssetsPath, "/") {
		opts.AssetsPath = opts.AssetsPath + "/"
	}

	client, err := fs.Sub(fsys, "client")
	if err != nil {
		panic(err)
	}

	renderer := render.New(fsys, opts.AssetsPath)
	assetsHandler := http.StripPrefix(opts.AssetsPath, fileServer(client))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), contextKey{}, &renderContext{
				renderer: renderer,
			})
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}, assetsHandler.ServeHTTP
}

func AddComponent(r *http.Request, component string) {
	rctx := getRenderContext(r)
	rctx.components = append(rctx.components, renderEntry{
		component: component,
	})
}

func Render(w io.Writer, r *http.Request) error {
	rctx := getRenderContext(r)

	keys := make([]string, len(rctx.components))
	i := 0
	for _, entry := range rctx.components {
		keys[i] = entry.component
		i++
	}

	return rctx.renderer.Render(w, keys...)
}

func Layout(component string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			AddComponent(r, component)
			next.ServeHTTP(w, r)
		})
	}
}

func Page(component string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddComponent(r, component)
		err := Render(w, r)
		if err != nil {
			// TODO
		}
	})
}
