package sveltigo

import (
	"net/http"

	"github.com/HazelnutParadise/sveltigo/render"
)

type contextKey struct{}

// RenderContext is used for lower level control over rendering.
// It allows direct access to the renderer and component slice.
type RenderContext struct {
	Renderer   *render.Renderer
	Components []render.Entry
	ErrPage    string

	csr    bool
	scdata render.SvelteContextData
}

// GetRenderContext returns the render context from the request, or nil if it doesn't exist.
func GetRenderContext(r *http.Request) *RenderContext {
	rctx, ok := r.Context().Value(contextKey{}).(*RenderContext)
	if !ok {
		return nil
	}

	return rctx
}

// MustGetRenderContext is like [GetRenderContext], but panics instead of returning nil.
func MustGetRenderContext(r *http.Request) *RenderContext {
	rctx := GetRenderContext(r)
	if rctx == nil {
		panic("golte middleware not registered")
	}

	return rctx
}

// Render renders all the components in the render context to the writer,
// with each subsequent component being a child of the previous.
func (r *RenderContext) Render(w http.ResponseWriter) {
	data := &render.RenderData{Entries: r.Components, ErrPage: r.ErrPage, SCData: r.scdata}
	err := r.Renderer.Render(w, data, r.csr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
