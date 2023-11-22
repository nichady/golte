package golte

import (
	"net/http"

	"github.com/nichady/golte/render"
)

type contextKey struct{}

type renderContext struct {
	renderer *render.Renderer
	layouts  []render.Entry
	errpage  string
}

func getRenderContext(r *http.Request) *renderContext {
	rctx, ok := r.Context().Value(contextKey{}).(*renderContext)
	if !ok {
		panic("golte middleware not registered")
	}

	return rctx
}
