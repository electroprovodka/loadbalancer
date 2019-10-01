package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func getServer(config *Config, router *http.ServeMux) (*http.Server, error) {
	// TODO: check other timeouts (header, idle, etc.)
	// TODO: Headers/Body size limit?
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", config.Port),
		// TODO: middlewares
		Handler:     router,
		ReadTimeout: time.Duration(config.ServerReadTimeout) * time.Second,
		// TODO: check correct value for this field https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
		WriteTimeout: time.Duration(config.ServerWriteTimeout) * time.Second,
		// TODO: Idle timeout for keep alive connections
	}
	return server, nil
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

	config, err := ReadConfig("config.yml")
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

	server, err := getServer(config, router)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
