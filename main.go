package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func parseFlags() string {
	var config string
	flag.StringVar(&config, "config", "", "path to the proxy config")
	flag.Parse()
	if config == "" {
		log.Fatal("--config is required field to start the server")
	}
	return config
}

func setupLogging() {
	// TODO: add setup

}

func getRouter(h http.HandlerFunc) *http.ServeMux {
	router := http.NewServeMux()
	// TODO: use TimeoutHandler for timeouts for the overall flow?
	router.HandleFunc("/", h)
	router.HandleFunc("/-/health", healthHandler)
	return router
}

func setupServerShutdown(server *http.Server) <-chan bool {
	done := make(chan bool)
	// TODO: original code contains size=1. Should we set it this way?
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		// Wait for system exit signal
		<-quit

		setServerHealthy(false)

		log.Warn("Shutting down the server")

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

	return done
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

	setupLogging()

	proxy, err := NewProxy(config)
	if err != nil {
		log.Fatal(err)
		return
	}

	router := getRouter(proxy.Handle)
	server, err := getServer(config, router, tracing, logging)
	if err != nil {
		log.Fatal(err)
		return
	}

	done := setupServerShutdown(server)

	log.Warnf("Starting the server on port :%d\n", config.Port)

	setServerHealthy(true)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Unexpected server error: %s\n", err)
	}

	// Wait until shutdown is finished
	<-done
	log.Warn("Server stopped")
}
