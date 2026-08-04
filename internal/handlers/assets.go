package handlers

import (
	"io/fs"

	webassets "buffetflow/web"
)

func mustStaticSub() fs.FS {
	return webassets.StaticFS()
}
