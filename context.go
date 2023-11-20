package golte

import (
	"net/http"

	"github.com/nichady/golte/render"
)

type contextKey struct{}

type renderContext = struct {
	renderer   *render.Renderer
	components []renderEntry
}

func getRenderContext(r *http.Request) *renderContext {
	rctx, ok := r.Context().Value(contextKey{}).(*renderContext)
	if !ok {
		panic("golte middleware not registered")
	}

	return rctx
}

type renderEntry struct {
	component string
	props     map[string]any
}
