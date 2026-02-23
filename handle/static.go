package handle

import (
	"embed"
	"io/fs"
	"net/http"
)

// StaticHandler serves an embedded static directory from within an embed.FS.
// For example, if you want to embed the `static` subdirectory, create an embed.FS:
//
// //go:embed static/*
// var staticFiles embed.FS
//
// Then call StaticHandler("static", "/static/", staticFiles)
//
// To serve the files from the "static" subdirectory at the route
// prefix "/static/". The slashes in the prefix ensure that files
// are routed to the correct path within the embed.FS.
func StaticHandler(dir, prefix string, files embed.FS) http.Handler {
	staticFS, err := fs.Sub(files, dir)
	if err != nil {
		panic(err)
	}

	var h http.Handler = http.FileServer(http.FS(staticFS))

	if len(prefix) > 0 {
		return http.StripPrefix(prefix, h)
	}
	return h
}
