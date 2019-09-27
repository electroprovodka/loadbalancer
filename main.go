package main

import (
	"fmt"
	"log"
	"net/http"
)

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

	// TODO: change HandleFunc?
	http.HandleFunc("/", proxy.handle)
	// TODO: use cusom handler
	err = http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
