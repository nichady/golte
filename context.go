package golte

import (
	"net/http"

	"github.com/nichady/golte/render"
)

type contextKey struct{}

type RenderContext struct {
	Renderer          *render.Renderer
	HandleRenderError func(*http.Request, []render.Entry, error)
	Layouts           []render.Entry
	ErrPage           string
}

func GetRenderContext(r *http.Request) *RenderContext {
	rctx, ok := r.Context().Value(contextKey{}).(*RenderContext)
	if !ok {
		panic("golte middleware not registered")
	}

	return rctx
}
