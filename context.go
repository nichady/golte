package golte

import (
	"net/http"

	"github.com/nichady/golte/render"
)

type contextKey struct{}

type renderContext = struct {
	renderer *render.Renderer
	entries  []render.Entry
}

func getRenderContext(r *http.Request) *renderContext {
	rctx, ok := r.Context().Value(contextKey{}).(*renderContext)
	if !ok {
		panic("golte middleware not registered")
	}

	return rctx
}
