package sveltigo

import (
	"io/fs"
	"net/http"
)

// fileServer is a wrapper around http.FileServer that won't do directory listing
func fileServer(fsys fs.FS) http.Handler {
	return http.FileServer(&wrapperfs{http.FS(fsys)})
}

type wrapperfs struct {
	fsys http.FileSystem
}

func (w *wrapperfs) Open(name string) (http.File, error) {
	file, err := w.fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			file.Close()
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, fs.ErrNotExist
	}

	return file, nil
}
