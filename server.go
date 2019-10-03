package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

type ProxyServer struct {
	server *http.Server
	router *http.ServeMux
	proxy  *Proxy
	// healthy is the marker of the server status
	// 0 means server is starting up or shutting down
	// 1 means server is up and running
	health int32

	done chan bool
}

func (p *ProxyServer) setServerHealth(h bool) {
	var v int32 = 0
	if h {
		v = 1
	}
	atomic.StoreInt32(&(p.health), v)
}

func (p *ProxyServer) healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&(p.health)) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// TODO: Authentication
func (p *ProxyServer) reloadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: read config path from request
		// TODO: reload server config (not only proxy)
		// TODO: maybe create new proxy instead of updating existing - possible memory leak?
		configPath := "config.yml"
		config, err := ReadConfig(configPath)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		setupLogging(config)
		err = p.proxy.Update(config)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
}

func (p *ProxyServer) setupServerShutdown() {
	p.done = make(chan bool)

	// TODO: original code contains size=1. Should we set it this way?
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Wait for system exit signal
		<-quit

		p.setServerHealth(false)

		log.Warn("Shutting down the server")

		// TODO: allow to setup shutdown wait
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Disable ongoing keep-alive connections
		p.server.SetKeepAlivesEnabled(false)
		if err := p.server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not shutdown the server: %s\n", err)
		}
		// Allow main goroutine to finish
		close(p.done)
	}()
}

func (p *ProxyServer) Start() {
	p.setupServerShutdown()
	p.setServerHealth(true)

	err := p.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Unexpected server error: %s\n", err)
	}

	// Wait until shutdown is finished
	<-p.done
}

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

func NewProxyServer(config *Config) (*ProxyServer, error) {
	p := ProxyServer{}

	proxy, err := NewProxy(config)
	if err != nil {
		return nil, err
	}
	p.proxy = proxy

	p.router = http.NewServeMux()
	// TODO: use TimeoutHandler for timeouts for the overall flow?
	p.router.HandleFunc("/", p.proxy.Handle)
	p.router.HandleFunc("/-/health", p.healthHandler())
	p.router.HandleFunc("/-/reload", p.reloadHandler())

	server, err := getServer(config, p.router, tracing)
	if err != nil {
		return nil, err
	}
	p.server = server
	return &p, nil
}
