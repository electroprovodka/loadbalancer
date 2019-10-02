package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/rs/xid"
)

// healthy is the marker of the server status
// 0 means server is starting up or shutting down
// 1 means server is up and running
var healthy int32

type key int

var requestIDKey key = 0

func getRequestID() string {
	return xid.New().String()
}

func trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = getRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r.Header.Set("X-Request-Id", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			requestID, ok := r.Context().Value(requestIDKey).(string)
			if !ok {
				// TODO: generate new ID?
				requestID = "unknown"
			}
			// TODO: log response code
			log.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
		}()
		next.ServeHTTP(w, r)
	})
}

func getServer(config *Config, router *http.ServeMux) (*http.Server, error) {
	// TODO: check other timeouts (header, idle, etc.)
	// TODO: Headers/Body size limit?
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", config.Port),
		// TODO: middlewares
		Handler:     trace(logging(router)),
		ReadTimeout: time.Duration(config.ServerReadTimeout) * time.Second,
		// TODO: check correct value for this field https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		WriteTimeout: time.Duration(config.ServerWriteTimeout) * time.Second,
		// TODO: Idle timeout for keep alive connections
	}
	return server, nil
}

func health(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt32(&healthy) == 1 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

func parseFlags() string {
	var config string
	flag.StringVar(&config, "config", "", "path to the proxy config")
	flag.Parse()
	if config == "" {
		log.Fatal("--config is required field to start the server")
	}
	return config
}

func main() {
	// TODO: multiple targets
	// TODO: redirect rules
	// TODO: file config
	// TODO: command line params
	// TODO: hot reload
	// TODO: logging
	// TODO: tests
	// TODO: graceful shutdown
	// TODO: signals processing
	// TODO: https
	// TODO: metrics?
	// TODO: healthchecks?
	// TODO: targets autodiscovery?
	// TODO: API for controlling
	// TODO: Docker image

	configPath := parseFlags()

	config, err := ReadConfig(configPath)
	if err != nil {
		log.Fatal(err)
		return
	}

	proxy, err := NewProxy(config)
	if err != nil {
		log.Fatal(err)
		return
	}

	router := http.NewServeMux()
	// TODO: use TimeoutHandler for timeouts for the overall flow?
	router.HandleFunc("/", proxy.Handle)
	router.HandleFunc("/-/health", health)

	server, err := getServer(config, router)
	if err != nil {
		log.Fatal(err)
		return
	}

	done := make(chan bool)
	// TODO: original code contains size=1. Should we set it this way?
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Wait for system exit signal
		<-quit

		atomic.StoreInt32(&healthy, 0)
		log.Println("Shutting down the server")

		// TODO: allow to setup shutdown wait
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Disable ongoing keep-alive connections
		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not shutdown the server: %s\n", err)
		}
		// Allow main goroutine to finish
		close(done)
	}()

	log.Printf("Starting the server on port :%d\n", config.Port)

	atomic.StoreInt32(&healthy, 1)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Unexpected server error: %s\n", err)
	}
	// Wait until shutdown is finished
	<-done
	log.Println("Server stopped")
}
