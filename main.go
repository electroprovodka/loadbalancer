package main

import (
	"flag"

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

	setupLogging(config)

	ps, err := NewProxyServer(config)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Warnf("Starting the server on port :%d\n", config.Port)
	ps.Start()
	log.Warn("Server stopped")
}
