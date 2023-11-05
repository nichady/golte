package golte

import (
	"context"
	"io"
	"io/fs"
	"net/http"

	"github.com/nichady/golte/render"
)

type golteKey struct{}

var key golteKey

type data struct {
	renderer   *render.Renderer
	components []string
}

func New(fsys fs.FS) func(http.Handler) http.Handler {
	renderer := render.New(fsys)

	sub, err := fs.Sub(fsys, "client")
	if err != nil {
		panic(err)
	}

	return func(next http.Handler) http.Handler {
		m := http.NewServeMux()
		m.Handle("/_golte/", http.StripPrefix("/_golte/", http.FileServer(http.FS(sub))))
		m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), key, &data{renderer: renderer}))
			next.ServeHTTP(w, r)
		})
		return m
	}
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

func getData(r *http.Request) *data {
	return r.Context().Value(key).(*data)
}

func AddComponent(r *http.Request, component string) {
	data := getData(r)
	data.components = append(data.components, component)
}

func Render(w io.Writer, r *http.Request) error {
	data := getData(r)
	return data.renderer.Render(w, data.components...)
}
