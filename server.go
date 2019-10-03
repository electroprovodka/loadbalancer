package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// healthy is the marker of the server status
// 0 means server is starting up or shutting down
// 1 means server is up and running
var healthy int32

func getServer(config *Config, router *http.ServeMux, middlewares ...Middleware) (*http.Server, error) {
	// TODO: check other timeouts (header, idle, etc.)
	// TODO: Headers/Body size limit?
	server := &http.Server{
		Addr:        fmt.Sprintf(":%d", config.Port),
		Handler:     applyMiddlewares(router, middlewares),
		ReadTimeout: time.Duration(config.ServerReadTimeout) * time.Second,
		// TODO: check correct value for this field https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		WriteTimeout: time.Duration(config.ServerWriteTimeout) * time.Second,
		// TODO: Idle timeout for keep alive connections
	}
	return server, nil
}

func setServerHealthy(h bool) {
	var v int32 = 0
	if h {
		v = 1
	}
	atomic.StoreInt32(&healthy, v)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&healthy) == 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}
