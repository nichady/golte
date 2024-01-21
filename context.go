package golte

import (
	"net/http"

	"github.com/nichady/golte/render"
)

type contextKey struct{}

// RenderContext is used for lower level control over rendering.
// It allows direct access to the renderer and component slice.
type RenderContext struct {
	Renderer   *render.Renderer
	Components []render.Entry
	ErrPage    string

	req    *http.Request
	scdata render.SvelteContextData
}

// GetRenderContext retrives the render context from the request.
func GetRenderContext(r *http.Request) *RenderContext {
	rctx, ok := r.Context().Value(contextKey{}).(*RenderContext)
	if !ok {
		panic("golte middleware not registered")
	}

	rctx.req = r
	return rctx
}

// Render renders all the components in the render context to the writer,
// with each subsequent component being a child of the previous.
func (r *RenderContext) Render(w http.ResponseWriter) {
	data := render.RenderData{Entries: r.Components, ErrPage: r.ErrPage, SCData: r.scdata}
	err := r.Renderer.Render(w, data, r.req.Header["Golte"] != nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
