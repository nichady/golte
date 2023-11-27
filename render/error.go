package render

import (
	"errors"
	"fmt"

	"github.com/dop251/goja"
)

func (r *Renderer) tryConvToRenderError(err error) error {
	ex, ok := err.(*goja.Exception)
	if !ok {
		return err
	}

	r.mtx.Lock()
	defer r.mtx.Unlock()

	if !r.renderfile.IsRenderError(ex.Value()) {
		return errors.New("golte error (this shouldn't happen): " + ex.String())
	}

	return &RenderError{
		Index:      int(ex.Value().ToObject(r.vm).Get("index").ToInteger()),
		StackTrace: ex.String(),
	}
}

type RenderError struct {
	Index      int // Index is the index of the node which caused the error
	StackTrace string
}

func (e *RenderError) Error() string {
	return fmt.Sprintf("render error occured at node %d: %s", e.Index, e.StackTrace)
}
