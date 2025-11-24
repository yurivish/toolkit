package handlers

import (
	"net/http"
	"sync"

	"github.com/starfederation/datastar-go/datastar"
)

var hotReloadOnlyOnce sync.Once

// hotreloadHandler is designed for the use case of a developer with a single
// browser tab open. That tab should have the following code (assuming this
// handler is running at /hot-reload):
//
// <div data-init="@get('/hot-reload', {retryMaxCount: 1000, retryInterval: 20, retryMaxWaitMs: 200})" id="hotreload"></div>
//
// When the server is shut down, this will attempt to reconnect until the server is restarted.
// That first successful reconnection will trigger the new server to send the reload script.
// After that initial reload, this handler will simply send an empty SSE connection thanks to
// the use of sync.Once.
//
// Todo: Make a version that will reconnect *all* clients – maybe by
func HotReloadHandler(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	hotReloadOnlyOnce.Do(func() {
		sse.ExecuteScript("window.location.reload()")
	})
	<-r.Context().Done()
}
