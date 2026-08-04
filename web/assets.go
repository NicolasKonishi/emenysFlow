package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func StaticFS() fs.FS {
	_, sourceFile, _, ok := runtime.Caller(0)
	directory := filepath.Join("web", "static")
	if ok {
		directory = filepath.Join(filepath.Dir(sourceFile), "static")
	}
	return os.DirFS(directory)
}
