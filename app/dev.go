// Package app: dev-only helpers behind the Dev flag (0007/0008).
//
// In development cozy-mug injects a tiny reload script into every rendered HTML
// page and serves a build id from /__cozy/dev-build. The script polls that
// endpoint; when the server process is replaced (by `cozy dev` or a plain
// restart) the build id changes and the browser refreshes itself. The endpoints
// are mounted only when Config.Dev is true, so production carries zero of this.
package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/lalithaar/cozy-mug/core"
)

const devReloadTag = `<script src="/__cozy/dev-reload"></script>`

func devInjectMiddleware(next core.Handler) core.Handler {
	return func(c *core.Context) core.Response {
		resp := next(c)
		if h, ok := resp.(core.HTML); ok {
			h.Body = injectDevReloadScript(h.Body)
			return h
		}
		return resp
	}
}

func injectDevReloadScript(body string) string {
	if i := strings.LastIndex(strings.ToLower(body), "</body>"); i >= 0 {
		return body[:i] + devReloadTag + body[i:]
	}
	return body + devReloadTag
}

var (
	devBuildIDOnce sync.Once
	devBuildIDVal  string
)

func devBuildID() string {
	devBuildIDOnce.Do(func() {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			devBuildIDVal = "dev"
			return
		}
		devBuildIDVal = hex.EncodeToString(b[:])
	})
	return devBuildIDVal
}

// devReloadHandler serves the dev script and the current build id. It is an
// http.Handler mounted under /__cozy/ so it bypasses the core.Response pipeline
// (and the injection above) and can set precise content types.
type devReloadHandler struct{}

func (devReloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "dev-reload", "/dev-reload":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, devReloadScript)
	case "dev-build", "/dev-build":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"build":%q}`, devBuildID())
	default:
		http.NotFound(w, r)
	}
}

const devReloadScript = `(function(){
  var key = "__cozy_dev_build";
  var url = "/__cozy/dev-build";
  async function tick() {
    try {
      var res = await fetch(url, {cache: "no-store"});
      if (!res.ok) return;
      var data = await res.json();
      var stored = null;
      try { stored = sessionStorage.getItem(key); } catch (e) {}
      if (stored !== data.build) {
        try { sessionStorage.setItem(key, data.build); } catch (e) {}
        if (stored !== null) {
          location.reload();
          return;
        }
      }
    } catch (e) {}
  }
  setInterval(tick, 750);
  tick();
})();`
