package golte

import (
	"io"
	"net/http"

	"github.com/nichady/golte/render"
)

type contextKey struct{}

// RenderContext is used for lower level control over rendering.
// It allows direct access to the renderer and component slice, and contains more detailed documentation.
type RenderContext struct {
	Renderer          *render.Renderer
	HandleRenderError func(*http.Request, []render.Entry, error)
	Components        []render.Entry
	ErrPage           string

	req *http.Request
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

// Render renders all the components in the render context to the writer.
// If an error occurs in rendering, it will call RenderErrorPage.
func (r *RenderContext) Render(w http.ResponseWriter) {
	err := r.Renderer.Render(w, r.Components, r.req.Header["Golte"] != nil)
	if err != nil {
		r.HandleRenderError(r.req, r.Components, err)
		if rerr, ok := err.(*render.RenderError); ok {
			r.Components = r.Components[:rerr.Index]
			r.RenderErrorPage(w, err.Error(), http.StatusInternalServerError)
		} else {
			// this shouldn't happen
			r.RenderFallback(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// RenderErrorPage adds the current error page to the component slice, then renders to the writer.
// If an error occurs while rendering the error page, the fallback error page is used instead.
// It will also write the status code to the header.
func (r *RenderContext) RenderErrorPage(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)

	page := render.Entry{Comp: r.ErrPage, Props: map[string]any{
		"message": message,
		"code":    status,
	}}
	r.Components = append(r.Components, page)

	err := r.Renderer.Render(w, r.Components, r.req.Header["Golte"] != nil)
	if err != nil {
		r.HandleRenderError(r.req, r.Components, err)
		r.RenderFallback(w, err.Error(), -1)
	}
}

// FenderFallback renders the fallback error.
func (r *RenderContext) RenderFallback(w http.ResponseWriter, message string, status int) {
	// TODO

	if status != -1 {
		w.WriteHeader(status)
	}
	io.WriteString(w, message)
}
