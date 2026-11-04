package server

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// RegisterUI serves the embedded React build. In development, if dist is empty,
// a small placeholder page is served so the binary still boots.
func RegisterUI(mux *http.ServeMux) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil || !hasIndex(sub) {
		mux.HandleFunc("/", servePlaceholder)
		return
	}

	fileServer := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(sub, path); err != nil {
			// SPA fallback
			f, openErr := sub.Open("index.html")
			if openErr != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.Copy(w, f)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func hasIndex(fsys fs.FS) bool {
	_, err := fs.Stat(fsys, "index.html")
	return err == nil
}

func servePlaceholder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Logka</title>
<style>
  :root { color-scheme: dark; --ink:#0c1210; --foam:#e8f2ec; --signal:#3dffa8; }
  body{margin:0;min-height:100vh;display:grid;place-items:center;
    background:radial-gradient(1200px 600px at 20% 0%,#163028,var(--ink));
    color:var(--foam);font-family:ui-sans-serif,system-ui,sans-serif}
  h1{font-size:clamp(3rem,8vw,6rem);letter-spacing:-.04em;margin:0}
  p{opacity:.7;max-width:28rem;text-align:center}
  code{color:var(--signal)}
</style></head>
<body>
  <main style="text-align:center;padding:2rem">
    <h1>Logka</h1>
    <p>UI assets not embedded yet. Build the frontend into <code>internal/server/dist</code> and rebuild the binary.</p>
  </main>
</body></html>`)
}
